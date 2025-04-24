// internal/service/reward_service_impl.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	zlog "github.com/rs/zerolog/log"
)

// rewardServiceImpl implements the RewardService interface.
type rewardServiceImpl struct {
	pool           *pgxpool.Pool
	rewardRepo     repository.RewardRepository
	userRewardRepo repository.UserRewardRepository
	pointRepo      repository.PointTransactionRepository
	userRelRepo    repository.UserRelationshipRepository
}

// Definisikan error spesifik untuk service layer jika perlu
var ErrInsufficientPoints = errors.New("insufficient points to claim reward")
var ErrInvalidReviewStatus = errors.New("invalid status provided for review")

// NewRewardService creates a new instance of RewardService.
func NewRewardService(
	pool *pgxpool.Pool,
	rewardRepo repository.RewardRepository,
	userRewardRepo repository.UserRewardRepository,
	pointRepo repository.PointTransactionRepository,
	userRelRepo repository.UserRelationshipRepository,
) RewardService {
	return &rewardServiceImpl{
		pool:           pool,
		rewardRepo:     rewardRepo,
		userRewardRepo: userRewardRepo,
		pointRepo:      pointRepo,
		userRelRepo:    userRelRepo,
	}
}

// ClaimReward implements the business logic for claiming a reward, including transaction management.
func (s *rewardServiceImpl) ClaimReward(ctx context.Context, childID int, rewardID int) (claimID int, err error) {
	// --- 1. Mulai Transaksi ---
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		zlog.Error().Err(err).Msg("Service: Failed to begin transaction for reward claim")
		return 0, fmt.Errorf("internal server error: could not start operation")
	}

	// --- 2. Defer Rollback/Commit ---
	defer func() {
		if p := recover(); p != nil {
			zlog.Error().Msgf("Service: Panic recovered during reward claim: %v", p)
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			zlog.Warn().Err(err).Int("child_id", childID).Int("reward_id", rewardID).Msg("Service: Rolling back transaction due to error during reward claim")
			rbErr := tx.Rollback(ctx)
			if rbErr != nil {
				zlog.Error().Err(rbErr).Msg("Service: Failed to rollback transaction")
			}
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				zlog.Error().Err(err).Int("child_id", childID).Int("reward_id", rewardID).Msg("Service: Failed to commit transaction for reward claim")
				err = fmt.Errorf("internal server error: could not finalize operation") // Set error jika commit gagal
			} else {
				zlog.Info().Int("claim_id", claimID).Msg("Service: Transaction committed successfully for reward claim")
			}
		}
	}()

	// --- 3. Logika Bisnis Inti dalam Transaksi ---

	// 3a. Dapatkan detail Reward
	rewardDetails, err := s.rewardRepo.GetRewardDetailsTx(ctx, tx, rewardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("reward_id", rewardID).Msg("Service: Reward not found for claim")
			err = fmt.Errorf("reward not found")
			return 0, err // Rollback
		}
		zlog.Error().Err(err).Int("reward_id", rewardID).Msg("Service: Error fetching reward details for claim")
		err = fmt.Errorf("internal server error: could not retrieve reward details")
		return 0, err // Rollback
	}

	// 3b. Validasi Hak Anak untuk Klaim Reward Ini
	childParents, err := s.userRelRepo.GetParentIDsByChildIDTx(ctx, tx, childID) // Metode Repo Baru (hanya return []int)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Service: Failed to get parents for child during claim validation")
		err = fmt.Errorf("%w: error checking child's parents", ErrInvitationFailed) // Atau error lain
		return 0, err                                                               // Rollback
	}
	isAllowedCreator := false
	for _, pID := range childParents {
		if pID == rewardDetails.CreatedByUserID { // Cek apakah pembuat reward adalah salah satu parent anak
			isAllowedCreator = true
			break
		}
	}
	if !isAllowedCreator {
		zlog.Warn().Int("child_id", childID).Int("reward_id", rewardID).Int("creator_id", rewardDetails.CreatedByUserID).Msg("Service: Child attempted to claim reward from unrelated parent")
		err = fmt.Errorf("forbidden: you cannot claim this reward")
		return 0, err // Rollback
	}

	// 3c. Dapatkan Poin Anak Saat Ini
	currentPoints, err := s.pointRepo.CalculateTotalPointsByUserIDTx(ctx, tx, childID)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Service: Error calculating child points for claim")
		err = fmt.Errorf("internal server error: could not retrieve points balance")
		return 0, err // Rollback
	}

	// 3d. Cek Poin Cukup
	if currentPoints < rewardDetails.RequiredPoints {
		zlog.Warn().Int("child_id", childID).Int("reward_id", rewardID).Int("current_points", currentPoints).Int("required_points", rewardDetails.RequiredPoints).Msg("Service: Insufficient points for reward claim")
		err = ErrInsufficientPoints // Gunakan error spesifik service
		return 0, err               // Rollback
	}

	// 3d. Buat Record UserReward (Klaim) dalam Transaksi
	// Asumsi: userRewardRepo.CreateClaimTx(ctx, tx, childID, rewardID, rewardDetails.RequiredPoints) -> (int, error) // return claimID
	// Anda perlu membuat metode ini di user_reward_repo.go
	claimID, err = s.userRewardRepo.CreateClaimTx(ctx, tx, childID, rewardID, rewardDetails.RequiredPoints)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Int("reward_id", rewardID).Msg("Service: Failed to create user reward claim within transaction")
		err = fmt.Errorf("internal server error: could not create claim record")
		return 0, err // Rollback
	}

	// 3e. Buat Transaksi Pengurangan Poin SEKARANG
	if rewardDetails.RequiredPoints > 0 { // Hanya kurangi jika poin > 0
		pointTx := &models.PointTransaction{
			UserID:          childID,
			ChangeAmount:    -rewardDetails.RequiredPoints, // Poin negatif
			TransactionType: models.TransactionTypeRedemption,
			// RelatedUserRewardID diisi NANTI setelah klaim dibuat? Atau bisa NULL?
			// Jika FK constraint mengizinkan NULL, biarkan 0 di sini. Jika tidak, perlu update lagi nanti.
			// Mari asumsikan bisa NULL/0 untuk sementara.
			// RelatedUserRewardID: 0,
			CreatedByUserID: childID,                                                            // Anak yang menginisiasi klaim
			Notes:           fmt.Sprintf("Points deducted for claiming reward ID %d", rewardID), // Opsional
		}
		err = s.pointRepo.CreateTransactionTx(ctx, tx, pointTx)
		if err != nil {
			zlog.Error().Err(err).Int("reward_id", rewardID).Int("child_id", childID).Msg("Service: Failed to create point deduction transaction within DB transaction")
			err = fmt.Errorf("internal server error: could not update points balance")
			return 0, err // Rollback
		}
		zlog.Info().Int("reward_id", rewardID).Int("points_deducted", rewardDetails.RequiredPoints).Int("child_id", childID).Msg("Service: Point deduction transaction created within DB transaction")
	}

	// 3f. Buat Record UserReward (Klaim) dalam Transaksi
	//    pointsDeducted di sini adalah nilai *snapshot* saat klaim, sesuai harga reward
	claimID, err = s.userRewardRepo.CreateClaimTx(ctx, tx, childID, rewardID, rewardDetails.RequiredPoints)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Int("reward_id", rewardID).Msg("Service: Failed to create user reward claim within transaction")
		// Jika pengurangan poin sudah terjadi, apakah perlu di-rollback manual?
		// Defer akan handle rollback DB, jadi state konsisten.
		err = fmt.Errorf("internal server error: could not create claim record")
		return 0, err // Rollback
	}

	// (Opsional) Update RelatedUserRewardID di PointTransaction jika FK tidak nullable
	// if !nullableFK && rewardDetails.RequiredPoints > 0 {
	//     err = s.pointRepo.UpdateRewardIDForTransactionTx(ctx, tx, /* ID transaksi poin */, claimID)
	//     if err != nil { /* rollback */ }
	// }

	return claimID, nil // Sukses
}

func (s *rewardServiceImpl) ReviewClaim(ctx context.Context, claimID int, parentID int, newStatus models.UserRewardStatus) (err error) {
	// Validasi input status (opsional tapi baik)
	if newStatus != models.UserRewardStatusApproved && newStatus != models.UserRewardStatusRejected {
		return ErrInvalidReviewStatus
	}

	// --- 1. Mulai Transaksi ---
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		zlog.Error().Err(err).Msg("Service: Failed to begin transaction for reward review")
		return fmt.Errorf("internal server error: could not start operation")
	}

	// --- 2. Defer Rollback/Commit ---
	defer func() {
		if p := recover(); p != nil {
			zlog.Error().Msgf("Service: Panic recovered during reward review: %v", p)
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			zlog.Warn().Err(err).Int("claim_id", claimID).Int("parent_id", parentID).Msg("Service: Rolling back transaction due to error during reward review")
			rbErr := tx.Rollback(ctx)
			if rbErr != nil {
				zlog.Error().Err(rbErr).Msg("Service: Failed to rollback transaction on review error")
			}
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				zlog.Error().Err(err).Int("claim_id", claimID).Int("parent_id", parentID).Msg("Service: Failed to commit transaction for reward review")
				err = fmt.Errorf("internal server error: could not finalize review")
			} else {
				zlog.Info().Int("claim_id", claimID).Str("final_status", string(newStatus)).Msg("Service: Transaction committed successfully for reward review")
			}
		}
	}()

	// --- 3. Logika Bisnis Inti dalam Transaksi ---

	// 3a. Dapatkan detail Klaim
	claimDetails, err := s.userRewardRepo.GetClaimDetailsForReviewTx(ctx, tx, claimID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("claim_id", claimID).Msg("Service: Claim not found for review")
			err = fmt.Errorf("reward claim not found")
			return err // Rollback
		}
		zlog.Error().Err(err).Int("claim_id", claimID).Msg("Service: Error fetching claim details for review")
		err = fmt.Errorf("internal server error: could not retrieve claim details")
		return err // Rollback
	}

	// 3b. Validasi Status Saat Ini
	if claimDetails.CurrentStatus != models.UserRewardStatusPending {
		zlog.Warn().Int("claim_id", claimID).Str("current_status", string(claimDetails.CurrentStatus)).Msg("Service: Review claim failed: Claim not in 'pending' status")
		err = fmt.Errorf("cannot review claim: current status is '%s', expected 'pending'", claimDetails.CurrentStatus)
		return err // Rollback
	}

	// 3c. Validasi Relasi Parent-Child
	isParentOfClaimant, err := s.userRelRepo.IsParentOfTx(ctx, tx, parentID, claimDetails.ChildID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", claimDetails.ChildID).Msg("Service: Error checking parent-child relationship during claim review")
		err = fmt.Errorf("internal server error: could not verify relationship")
		return err // Rollback
	}
	if !isParentOfClaimant {
		zlog.Warn().Int("claim_id", claimID).Int("parent_id", parentID).Int("child_id", claimDetails.ChildID).Msg("Service: Review claim failed: Requesting user is not the parent")
		err = fmt.Errorf("forbidden: you are not authorized to review claims for this child")
		return err // Rollback
	}

	canReview := false
	if parentID == claimDetails.RewardCreatorID {
		canReview = true // Reviewer adalah pembuat reward
	} else {
		// Cek apakah reviewer punya anak bersama dengan pembuat reward
		hasShared, errShared := s.userRelRepo.HasSharedChildTx(ctx, tx, parentID, claimDetails.RewardCreatorID) // Perlu metode Tx ini
		if errShared != nil {
			zlog.Error().Err(errShared).Int("reviewer", parentID).Int("creator", claimDetails.RewardCreatorID).Msg("Service: Error checking shared child for reward review")
			err = fmt.Errorf("%w: error checking reviewer permissions", ErrInvitationFailed) // atau error lain
			return err                                                                       // Rollback
		}
		if hasShared {
			canReview = true // Boleh review karena satu "keluarga"
		}
	}
	if !canReview {
		zlog.Warn().Int("claim_id", claimID).Int("reviewer", parentID).Int("creator", claimDetails.RewardCreatorID).Msg("Service: Parent attempted to review claim for reward created by unrelated parent")
		err = fmt.Errorf("forbidden: you cannot review claims for rewards created outside your family scope")
		return err // Rollback
	}

	// 3d. Update Status Klaim dalam Transaksi
	err = s.userRewardRepo.UpdateClaimStatusTx(ctx, tx, claimID, newStatus, parentID)
	if err != nil {
		// Error bisa karena status sudah berubah (concurrency) atau error DB lain
		zlog.Error().Err(err).Int("claim_id", claimID).Str("new_status", string(newStatus)).Msg("Service: Failed to update claim status within transaction")
		// Kembalikan error asli dari repo jika informatif (misal: "current status is already 'approved'")
		// Jika tidak, bungkus dengan pesan generik
		if strings.Contains(err.Error(), "current status is already") {
			return err // Kembalikan error status change
		}
		err = fmt.Errorf("internal server error: could not update claim status")
		return err // Rollback
	}

	// 3e. Jika Approved, Buat Transaksi Pengurangan Poin
	// --- MODIFIKASI: HAPUS BLOK PENGURANGAN POIN SAAT APPROVE ---
	// if newStatus == models.UserRewardStatusApproved { /* ... kode pengurangan poin dihapus ... */ }

	// --- TAMBAHKAN BLOK PENGEMBALIAN POIN SAAT REJECT ---
	if newStatus == models.UserRewardStatusRejected {
		// Hanya kembalikan poin jika memang ada poin yang tercatat untuk dikurangi
		if claimDetails.PointsDeducted > 0 {
			refundTx := &models.PointTransaction{
				UserID:              claimDetails.ChildID,
				ChangeAmount:        claimDetails.PointsDeducted, // Poin POSITIF
				TransactionType:     models.TransactionTypeManualAdjustment, // Atau tipe baru: "reward_refund"
				RelatedUserRewardID: claimID,                   // Kaitkan dengan klaim yang ditolak
				CreatedByUserID:     parentID,                    // Parent yang reject
				Notes:               fmt.Sprintf("Points refunded for rejected reward claim ID %d", claimID),
			}
			err = s.pointRepo.CreateTransactionTx(ctx, tx, refundTx)
			if err != nil {
				zlog.Error().Err(err).Int("claim_id", claimID).Msg("Service: Failed to create point refund transaction after claim rejection")
				// Ini masalah serius, tapi status klaim sudah 'rejected'. Tetap rollback?
				// Mungkin lebih baik biarkan commit status reject tapi log error refund.
				// Atau kembalikan error internal. Kita pilih rollback untuk konsistensi.
				err = fmt.Errorf("internal server error: claim rejected but failed to refund points")
				return err // Rollback
			}
			zlog.Info().Int("claim_id", claimID).Int("points_refunded", claimDetails.PointsDeducted).Int("child_id", claimDetails.ChildID).Msg("Service: Point refund transaction created within DB transaction")
		} else {
            zlog.Info().Int("claim_id", claimID).Msg("Service: Claim rejected, no points to refund (PointsDeducted was 0)")
        }
	}

	return nil // Sukses
}
