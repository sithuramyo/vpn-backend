package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vpn-backend/internal/auth"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
)

// ErrNotAuthorizedGoogleAccount is returned whenever a Google identity does
// not correspond to a pre-provisioned, ACTIVE administrator. The backend
// deliberately never auto-creates an admin row from an arbitrary Google
// account — an operator must provision it first (see scripts/seed-admin.sql).
var ErrNotAuthorizedGoogleAccount = errors.New("google account is not authorized to access this system")

type AuthService struct {
	verifier *auth.GoogleVerifier
	sessions *auth.SessionManager
	admins   *repositories.AdminRepository
	audit    *AuditService
}

func NewAuthService(verifier *auth.GoogleVerifier, sessions *auth.SessionManager, admins *repositories.AdminRepository, audit *AuditService) *AuthService {
	return &AuthService{verifier: verifier, sessions: sessions, admins: admins, audit: audit}
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Admin     *models.Admin
}

// LoginWithGoogle verifies the raw Google ID token, looks up the matching
// admin purely by google_sub, and rejects anything that is not an
// already-provisioned, ACTIVE admin. On success it issues this backend's
// own session token — the frontend must never be trusted to supply a role
// or status itself.
func (s *AuthService) LoginWithGoogle(idToken, ipAddress string) (*LoginResult, error) {
	identity, err := s.verifier.Verify(idToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	admin, err := s.admins.FindByGoogleSub(identity.Sub)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotAuthorizedGoogleAccount
		}
		return nil, err
	}

	if admin.Status != models.AdminStatusActive {
		return nil, ErrNotAuthorizedGoogleAccount
	}

	// Keep the cached profile fresh (name/picture can change on Google's side).
	admin.Name = identity.Name
	admin.PictureURL = identity.Picture
	admin.Email = identity.Email
	if err := s.admins.Update(admin); err != nil {
		return nil, err
	}
	_ = s.admins.TouchLastLogin(admin.ID)

	token, expiresAt, err := s.sessions.Issue(admin.ID)
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		_ = s.audit.Record(&admin.ID, models.ActionAdminLogin, "admin", &admin.ID, ipAddress, map[string]any{
			"email": admin.Email,
		})
	}

	return &LoginResult{Token: token, ExpiresAt: expiresAt, Admin: admin}, nil
}

func (s *AuthService) GetByID(id uuid.UUID) (*models.Admin, error) {
	return s.admins.FindByID(id)
}
