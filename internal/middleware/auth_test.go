package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/auth"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/testutil"
)

func setupRouter(t *testing.T) (*gin.Engine, *auth.SessionManager, *repositories.AdminRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	repos := repositories.New(db)
	sessions := auth.NewSessionManager("test-secret", time.Hour)

	r := gin.New()
	authed := r.Group("/")
	authed.Use(middleware.RequireAuth(sessions, repos.Admins))
	authed.GET("/whoami", func(c *gin.Context) {
		c.JSON(200, gin.H{"role": middleware.CurrentAdmin(c).Role})
	})
	authed.GET("/admin-only", middleware.RequireRole(models.AdminRoleAdmin), func(c *gin.Context) {
		c.Status(200)
	})

	return r, sessions, repos.Admins
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	r, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthAcceptsActiveAdmin(t *testing.T) {
	r, sessions, admins := setupRouter(t)

	admin := &models.Admin{GoogleSub: "sub-1", Email: "a@example.com", Role: models.AdminRoleViewer, Status: models.AdminStatusActive}
	if err := admins.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, _, err := sessions.Issue(admin.ID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireAuthRejectsDisabledAdmin(t *testing.T) {
	r, sessions, admins := setupRouter(t)

	admin := &models.Admin{GoogleSub: "sub-2", Email: "b@example.com", Role: models.AdminRoleViewer, Status: models.AdminStatusDisabled}
	if err := admins.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, _, err := sessions.Issue(admin.ID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403 for a disabled admin, got %d", w.Code)
	}
}

func TestRequireRoleRejectsInsufficientRole(t *testing.T) {
	r, sessions, admins := setupRouter(t)

	admin := &models.Admin{GoogleSub: "sub-3", Email: "c@example.com", Role: models.AdminRoleViewer, Status: models.AdminStatusActive}
	if err := admins.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, _, err := sessions.Issue(admin.ID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403 for a VIEWER hitting an ADMIN-only route, got %d", w.Code)
	}
}

func TestRequireRoleAllowsCorrectRole(t *testing.T) {
	r, sessions, admins := setupRouter(t)

	admin := &models.Admin{GoogleSub: "sub-4", Email: "d@example.com", Role: models.AdminRoleAdmin, Status: models.AdminStatusActive}
	if err := admins.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, _, err := sessions.Issue(admin.ID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for an ADMIN hitting an ADMIN-only route, got %d", w.Code)
	}
}
