package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
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
