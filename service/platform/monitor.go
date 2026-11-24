package platform

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/monitor"
	"dst-management-platform-api/service"
	"strings"
)

// MonitorCurrentData 是当前监控快照的响应。Available=false 表示采集器
// 尚未完成第一次 5 秒采样，不是接口错误。
type MonitorCurrentData struct {
	Available       bool              `json:"available"`
	Status          string            `json:"status"`
	IntervalSeconds int               `json:"intervalSeconds"`
	Snapshot        *monitor.Snapshot `json:"snapshot,omitempty"`
}

func (s *Service) MonitorCurrent() service.Result {
	snapshot := monitor.Current()
	data := MonitorCurrentData{
		Available:       snapshot != nil,
		Status:          "collecting",
		IntervalSeconds: monitor.IntervalSeconds,
	}
	if snapshot != nil {
		data.Snapshot = snapshot
		data.Status = "ok"
		if len(snapshot.Warnings) > 0 {
			data.Status = "degraded"
		}
	}
	return service.Success(data)
}

func (s *Service) MonitorHistory(rangeName string) service.Result {
	rangeName = strings.TrimSpace(rangeName)
	if rangeName == "" {
		rangeName = "1h"
	}
	data, err := monitor.History(rangeName)
	if err != nil {
		return service.Fail(constant.CodePlatformBadRequest)
	}
	return service.Success(data)
}

func (s *Service) MonitorProcesses(sortBy, order, query string, limit int) service.Result {
	sortBy = strings.TrimSpace(sortBy)
	order = strings.TrimSpace(order)
	if sortBy == "" {
		sortBy = "cpu"
	}
	if order == "" {
		order = "desc"
	}
	if sortBy != "cpu" && sortBy != "memory" && sortBy != "name" {
		return service.Fail(constant.CodePlatformBadRequest)
	}
	if order != "asc" && order != "desc" {
		return service.Fail(constant.CodePlatformBadRequest)
	}
	if limit < 0 || limit > 500 {
		return service.Fail(constant.CodePlatformBadRequest)
	}
	if len(query) > 128 {
		query = query[:128]
	}
	return service.Success(monitor.QueryProcesses(sortBy, order, query, limit))
}
