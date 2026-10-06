package models

import "testing"

func TestAnalyticsGroup_TableName(t *testing.T) {
	if (AnalyticsGroup{}).TableName() != "analytics_groups" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsGraph_TableName(t *testing.T) {
	if (AnalyticsGraph{}).TableName() != "analytics_graphs" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsFilter_TableName(t *testing.T) {
	if (AnalyticsFilter{}).TableName() != "analytics_filters" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsCustomFilter_TableName(t *testing.T) {
	if (AnalyticsCustomFilter{}).TableName() != "analytics_custom_filters" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsGroupFilter_TableName(t *testing.T) {
	if (AnalyticsGroupFilter{}).TableName() != "analytics_group_filters" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsUserGroup_TableName(t *testing.T) {
	if (AnalyticsUserGroup{}).TableName() != "analytics_user_groups" {
		t.Error("wrong table name")
	}
}

func TestAnalyticsCategoryConstants(t *testing.T) {
	if AnalyticsCategoryOverview != "overview" {
		t.Error("wrong constant")
	}
	if AnalyticsCategoryPayments != "payments" {
		t.Error("wrong constant")
	}
}

func TestAnalyticsChartTypeConstants(t *testing.T) {
	if AnalyticsChartTypeLine != "line" {
		t.Error("wrong constant")
	}
}
