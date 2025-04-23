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
	// Asumsi ada metode repo baru yang menerima Tx

	// 3a. Dapatkan detail Reward (terutama required_points)
	// Asumsi: rewardRepo.GetRewardDetailsTx(ctx, tx, rewardID) -> (*RewardDetails, error)
	//         type RewardDetails struct { RequiredPoints int; /* ... */ }
	// Anda perlu membuat metode ini di reward_repo.go
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

	// 3b. Dapatkan Poin Anak Saat Ini
	// Asumsi: pointRepo.CalculateTotalPointsByUserIDTx(ctx, tx, childID) -> (int, error)
	// Anda perlu membuat metode ini di point_transaction_repo.go
	currentPoints, err := s.pointRepo.CalculateTotalPointsByUserIDTx(ctx, tx, childID)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Service: Error calculating child points for claim")
		err = fmt.Errorf("internal server error: could not retrieve points balance")
		return 0, err // Rollback
	}

	// 3c. Cek Poin Cukup
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

	// Klaim berhasil dibuat (status 'pending'), poin *belum* dikurangi.
	// Pengurangan poin terjadi saat parent *menyetujui* klaim (ini akan ada di metode service lain, misal ReviewClaim).

	// Jika ingin poin langsung dikurangi saat klaim (bukan saat approval), tambahkan langkah 3e:
	/*
	   // 3e. Buat Transaksi Poin Negatif dalam Transaksi DB
	   pointTx := &models.PointTransaction{
	       UserID:              childID,
	       ChangeAmount:        -rewardDetails.RequiredPoints, // Poin negatif
	       TransactionType:     models.TransactionTypeRedemption,
	       RelatedUserRewardID: claimID, // Gunakan ID klaim yang baru dibuat
	       CreatedByUserID:     childID, // Anak yang melakukan klaim
	   }
	   err = s.pointRepo.CreateTransactionTx(ctx, tx, pointTx)
	   if err != nil {
	       zlog.Error().Err(err).Int("claim_id", claimID).Msg("Service: Failed to create point deduction transaction within DB transaction")
	       err = fmt.Errorf("internal server error: could not update points balance")
	       return 0, err // Rollback
	   }
	   zlog.Info().Int("claim_id", claimID).Int("points_deducted", rewardDetails.RequiredPoints).Int("child_id", childID).Msg("Service: Point deduction transaction created within DB transaction")
	*/

	// Jika semua berhasil, err = nil, defer akan commit
	return claimID, nil // Sukses
}

func (s *rewardServiceImpl) ReviewClaim(ctx context.Context, claimID int, parentID int, newStatus models.UserRewardStatus) (err error) {
	// Validasi input status (opsional tapi baik)
	if newStatus != models.UserRewardStatusApproved && newStatus != models.UserRewardStatusRejected {
		return fmt.Errorf("invalid target status for review: %s", newStatus)
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
	// Asumsi ada metode repo baru yang menerima Tx

	// 3a. Dapatkan detail Klaim (childID, status saat ini, pointsDeducted)
	// Asumsi: userRewardRepo.GetClaimDetailsForReviewTx(ctx, tx, claimID) -> (*ClaimReviewDetails, error)
	//          type ClaimReviewDetails struct { ChildID int; CurrentStatus models.UserRewardStatus; PointsDeducted int; }
	// Anda perlu membuat metode ini di user_reward_repo.go
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
	// Asumsi: userRelRepo.IsParentOfTx(ctx, tx, parentID, childID) -> (bool, error)
	// Metode ini sudah Anda buat implementasinya di user_relationship_repo.go
	isParent, err := s.userRelRepo.IsParentOfTx(ctx, tx, parentID, claimDetails.ChildID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", claimDetails.ChildID).Msg("Service: Error checking parent-child relationship during claim review")
		err = fmt.Errorf("internal server error: could not verify relationship")
		return err // Rollback
	}
	if !isParent {
		zlog.Warn().Int("claim_id", claimID).Int("parent_id", parentID).Int("child_id", claimDetails.ChildID).Msg("Service: Review claim failed: Requesting user is not the parent")
		err = fmt.Errorf("forbidden: you are not authorized to review claims for this child")
		return err // Rollback
	}

	// 3d. Update Status Klaim dalam Transaksi
	// Asumsi: userRewardRepo.UpdateClaimStatusTx(ctx, tx, claimID, newStatus, parentID) -> error
	// Metode ini sudah Anda buat implementasinya di user_reward_repo.go
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
	if newStatus == models.UserRewardStatusApproved {
		if claimDetails.PointsDeducted > 0 {
			pointTx := &models.PointTransaction{
				UserID:              claimDetails.ChildID,
				ChangeAmount:        -claimDetails.PointsDeducted, // Poin negatif
				TransactionType:     models.TransactionTypeRedemption,
				RelatedUserRewardID: claimID,
				CreatedByUserID:     parentID, // Parent yang approve
			}
			// Asumsi: pointRepo.CreateTransactionTx(ctx, tx, pointTx) -> error
			// Metode ini sudah Anda buat implementasinya di point_transaction_repo.go
			err = s.pointRepo.CreateTransactionTx(ctx, tx, pointTx)
			if err != nil {
				zlog.Error().Err(err).Int("claim_id", claimID).Msg("Service: Failed to create point deduction transaction after claim approval")
				err = fmt.Errorf("internal server error: could not update points balance")
				return err // Rollback
			}
			zlog.Info().Int("claim_id", claimID).Int("points_deducted", claimDetails.PointsDeducted).Int("child_id", claimDetails.ChildID).Msg("Service: Point deduction transaction created within DB transaction")
		} else {
			zlog.Info().Int("claim_id", claimID).Msg("Service: Claim approved, but no points deducted (PointsDeducted <= 0)")
		}
	}

	// Jika semua berhasil, err = nil, defer akan commit
	return nil // Sukses
}
