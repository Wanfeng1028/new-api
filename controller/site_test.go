package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSiteModeControllerTest(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:site_mode_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	origDB := model.DB
	origLOGDB := model.LOG_DB
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.Option{}))

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	origSiteModeValue, origSiteModeExists := common.OptionMap[model.SiteModeOptionKey]
	common.OptionMap[model.SiteModeOptionKey] = model.SiteModeBeta
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if origSiteModeExists {
			common.OptionMap[model.SiteModeOptionKey] = origSiteModeValue
		} else {
			delete(common.OptionMap, model.SiteModeOptionKey)
		}
		common.OptionMapRWMutex.Unlock()

		model.DB = origDB
		model.LOG_DB = origLOGDB
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
}

func TestUpdateSiteModeSwitchesModeAndPersists(t *testing.T) {
	setupSiteModeControllerTest(t)

	bodyBytes, err := common.Marshal(map[string]string{"mode": model.SiteModeBetaPlus})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/site-mode", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	UpdateSiteMode(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"success":true`)
	assert.Contains(t, w.Body.String(), `"mode":"beta_plus"`)
	assert.Equal(t, model.SiteModeBetaPlus, model.GetSiteMode())

	var opt model.Option
	err = model.DB.Where("`key` = ?", model.SiteModeOptionKey).First(&opt).Error
	require.NoError(t, err)
	assert.Equal(t, model.SiteModeBetaPlus, opt.Value)

	w2 := httptest.NewRecorder()
	ctx2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodGet, "/api/site-mode", nil)
	ctx2.Request = req2

	GetSiteMode(ctx2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"mode":"beta_plus"`)
}

func TestUpdateSiteModeRejectsInvalidMode(t *testing.T) {
	setupSiteModeControllerTest(t)

	bodyBytes, err := common.Marshal(map[string]string{"mode": "gamma"})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/site-mode", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	UpdateSiteMode(ctx)

	assert.Contains(t, w.Body.String(), `"success":false`)
	assert.Contains(t, w.Body.String(), "Site mode must be beta, beta_plus, or release")
	assert.Equal(t, model.SiteModeBeta, model.GetSiteMode())

	var count int64
	err = model.DB.Model(&model.Option{}).Where("`key` = ?", model.SiteModeOptionKey).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestUpdateSiteModeRejectsMalformedJSON(t *testing.T) {
	setupSiteModeControllerTest(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/site-mode", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	UpdateSiteMode(ctx)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, model.SiteModeBeta, model.GetSiteMode())
}

// TestUpdateSiteModeWithSessionBearerToken exercises the exact credential
// form the official website's bridge presents: the access token issued for
// a dashboard login session, sent as a Bearer header through RootAuth.
// Before this test existed, the handler was only tested in isolation and
// the bridge-to-RootAuth contract went unverified.
func TestUpdateSiteModeWithSessionBearerToken(t *testing.T) {
	setupSiteModeControllerTest(t)

	require.NoError(t, model.DB.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserAccessToken{}, &model.AuthFlow{}, &model.AuditLog{}))

	root := &model.User{
		Username:    "site-mode-root",
		AffCode:     "site-mode-root-aff",
		Password:    "unused-in-test",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
	}
	require.NoError(t, model.DB.Create(root).Error)
	plain := &model.User{
		Username:    "site-mode-plain",
		AffCode:     "site-mode-plain-aff",
		Password:    "unused-in-test",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
	}
	require.NoError(t, model.DB.Create(plain).Error)

	rootBundle, err := service.CreateLoginSession(root.Id, "password", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotEmpty(t, rootBundle.AccessToken)
	plainBundle, err := service.CreateLoginSession(plain.Id, "password", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	router := gin.New()
	router.POST("/api/site-mode", middleware.RootAuth(), UpdateSiteMode)
	router.GET("/api/site-mode", GetSiteMode)

	post := func(token string) *httptest.ResponseRecorder {
		bodyBytes, err := common.Marshal(map[string]string{"mode": model.SiteModeBetaPlus})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/site-mode", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// No credential at all: rejected, mode untouched.
	w := post("")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, model.SiteModeBeta, model.GetSiteMode())

	// Garbage bearer value: rejected, mode untouched.
	w = post("not-a-real-token")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, model.SiteModeBeta, model.GetSiteMode())

	// A common user's session token passes authentication but not RootAuth.
	w = post(plainBundle.AccessToken)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, model.SiteModeBeta, model.GetSiteMode())

	// The root session token — the bridge's credential form — switches the
	// mode and the public read reports the new value.
	w = post(rootBundle.AccessToken)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"mode":"beta_plus"`)
	assert.Equal(t, model.SiteModeBetaPlus, model.GetSiteMode())

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/site-mode", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"mode":"beta_plus"`)
}
