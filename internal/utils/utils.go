package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetPaginationParams(c *gin.Context) (int, int) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
    
    if page < 1 { page = 1 }
    if pageSize < 1 || pageSize > 100 { pageSize = 10 }
    
    return page, pageSize
}