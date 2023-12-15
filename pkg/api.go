package pkg

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pangpanglabs/echoswagger/v2"
)

func (arc *ArcStorage) handlerWrapper(selfServiceName string, h echo.HandlerFunc) echo.HandlerFunc {
	if arc.gossipKVCache != nil {
		return arc.gossipKVCache.SensorIDHandlerWrapper(selfServiceName, h, true)
	}
	return h
}

// SetupWeb Set interface
func (arc *ArcStorage) SetupWeb(root echoswagger.ApiRoot, base, selfServiceName string) {

	g := root.Group(ArcStorageAPI, base)
	g.GET("/sensorids", arc.handlerWrapper(selfServiceName, arc.getSensorIDs)).
		AddParamQuery(true, "inside", "inside swarm or not", false).
		AddResponse(http.StatusOK, `
		{
			"code": 0,
			"msg": "OK",
			"data": [
				"A00000000000"
			]
		}
		`, SensorIDResponse{}, nil).
		AddResponse(http.StatusNotFound, `
		{
			"code": 404,
			"msg": "Not Found"
		}		
		`, nil, nil).
		AddResponse(http.StatusTooManyRequests, `
		{
			"code": 429,
			"msg": "Too Many Requests:"+ id
		}		
		`, nil, nil).
		AddResponse(http.StatusServiceUnavailable, `
		{
			"code":503,
			"msg":"Service Unavailable"
		}	
		`, nil, nil).
		SetOperationId("sensorids").
		SetSummary("Get information of sensor ids")

	g = root.Group(TDEngineAPI, base)
	g.GET("/arc", arc.handlerWrapper(selfServiceName, arc.getSensorLists)).
		AddParamQuery(true, "inside", "inside swarm or not", false).
		AddParamQuery("", "sensorids", "多个ID逗号分隔", true).
		AddParamQuery(int64(0), "from", "起始时间", true).
		AddParamQuery(int64(0), "to", "终止时间", true).
		AddParamQuery("", "function", "可选项,聚合查询", false).
		AddParamQuery("", "interval", "可选项,聚合时间段的窗口", false).
		AddParamQuery("", "fill", "可选项,数据填充格式", false).
		AddResponse(http.StatusOK, `
		- 可选项说明: SQL查询使用函数(聚合函数、选择函数、计算函数、按窗口切分聚合等)。
		- 不使用可选项，则输出查询到的所有数据。
		- function - 单个输出选择函数,推荐first, 参数:avg,sum,min,max,first,last。
		- interval - 聚合时间段的窗口,interval指定,最短时间间隔10毫秒(10a),推荐100ms。
		- fill     - 指定某一窗口区间数据缺失的情况下的填充模式,推荐使用PREV,详细查看文档。
		{
			"code": 0,
			"msg": "OK",
			"data": [
				{
					"sensorid": "A00000000000",
					"data": [
						{
							"Time": 1658209829000,
							"arc": []byte{0x94,0xC9,0x60,0x00,0xC2,0x48}]
						},
					"count": 1
				}
			]
		}
		`, []byte{}, nil).
		AddResponse(http.StatusBadRequest, `
		{
			"code": 400,
			"msg": "Bad Request"
		}		
		`, nil, nil).
		AddResponse(http.StatusServiceUnavailable, `
		{
			"code":503,
			"msg":"Request Timeout"
		}	
		`, nil, nil).
		AddResponse(http.StatusNotFound, `
		{
			"code": 404,
			"msg": "Not Found"
		}		
		`, nil, nil).
		AddResponse(http.StatusTooManyRequests, `
		{
			"code": 429,
			"msg": "Too Many Requests:"+ id
		}		
		`, nil, nil).
		SetOperationId("arc").
		SetSummary("Return arc values. Querying from TDEngine")

}
