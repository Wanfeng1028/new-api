package controller

import (
	"net/http"

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
