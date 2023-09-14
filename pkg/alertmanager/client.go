package alertmanager

import (
	"context"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	alertapi "github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"k8s.io/klog/v2"
)

var logger klog.Logger = klog.LoggerWithName(klog.Background(), "alertmanager")

type AlertManager struct {
	Host    string
	ApiPath string
}

func New(host string, apiPath string) *AlertManager {
	if apiPath == "" {
		apiPath = API_PATH
	}

	return &AlertManager{
		Host:    host,
		ApiPath: apiPath,
	}
}

func (alertmanager *AlertManager) Alert(alertInfo *AlertInfo) {
	alert := alertmanager.createAlert(alertInfo)

	response, err := alertmanager.sendAlertToAlertmanager(alert)
	if err != nil {
		logger.Error(err, "Alert manager error")
		return
	}

	logger.Info("Response from alertmanager", "response", response)
}

func (alertmanager *AlertManager) createAlert(alertInfo *AlertInfo) *models.PostableAlert {
	alert := &models.PostableAlert{
		Annotations: map[string]string{
			"description": alertInfo.Description,
		},
		Alert: models.Alert{
			Labels: map[string]string{
				"alertname": alertInfo.Name,
				"severity":  alertInfo.Severity,
				"resource":  alertInfo.Resource,
				"instance":  alertInfo.Instance,
				"namespace": alertInfo.Namespace,
			},
		},
		StartsAt: strfmt.DateTime(time.Now().UTC()),
		//EndsAt:   strfmt.DateTime(time.Now().Add(time.Hour).UTC()),
	}

	return alert
}

func (alertmanager *AlertManager) sendAlertToAlertmanager(alert *models.PostableAlert) (*alertapi.PostAlertsOK, error) {
	transport := httptransport.New(alertmanager.Host, alertmanager.ApiPath, nil)
	alertmanagerClient := client.New(transport, nil)

	postAlertsParams := alertapi.PostAlertsParams{
		Alerts:  []*models.PostableAlert{alert},
		Context: context.Background(),
	}

	response, err := alertmanagerClient.Alert.PostAlerts(&postAlertsParams)
	if err != nil {
		return nil, err
	}

	return response, nil
}
