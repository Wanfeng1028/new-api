package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetSiteMode publicly reports the site's current release stage (beta,
// beta_plus, or release) so the official website can adapt what it shows.
// Admins change it through the SiteMode option.
func GetSiteMode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"mode": model.GetSiteMode(),
		},
	})
}

// UpdateSiteMode switches the site's release stage. It is the write side
// of GetSiteMode, restricted to root users like the settings API that
// writes the same option; the official website reaches it through its
// bridge with a root session token. The value goes through the SiteMode
// option validation in model.UpdateOption.
func UpdateSiteMode(c *gin.Context) {
	var request struct {
		Mode string `json:"mode"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
		return
	}
	if err := model.UpdateOption(model.SiteModeOptionKey, request.Mode); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"mode": model.GetSiteMode(),
		},
	})
}

// GetConnectedModels publicly lists the models covered by enabled
// channels. The official website uses it to filter its model library in
// beta_plus and release modes.
func GetConnectedModels(c *gin.Context) {
	models := model.GetConnectedModels()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"models": models,
			"count":  len(models),
		},
	})
}
