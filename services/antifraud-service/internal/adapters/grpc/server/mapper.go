package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

var verdictToProto = map[models.Verdict]antifraudv1.Verdict{
	models.VERDICT_ALLOW: antifraudv1.Verdict_VERDICT_ALLOW,
	models.VERDICT_DENY:  antifraudv1.Verdict_VERDICT_DENY,
}

func ruleToProto(rule *models.RuleConfig) *antifraudv1.Rule {
	out := &antifraudv1.Rule{
		Code:        rule.Code,
		Description: rule.Description,
		Enabled:     rule.Enabled,
		Params: &antifraudv1.RuleParams{
			Limits:        make(map[string]int64, len(rule.Params.Limits)),
			MaxCount:      int32(rule.Params.MaxCount),
			WindowSeconds: int32(rule.Params.WindowSeconds),
		},
		UpdatedAt: timestamppb.New(rule.UpdatedAt),
	}

	for currency, limit := range rule.Params.Limits {
		out.Params.Limits[string(currency)] = limit
	}

	if rule.UpdatedBy != nil {
		out.UpdatedBy = rule.UpdatedBy.String()
	}

	return out
}

// paramsFromProto: nil - пороги не меняются; коды валют приводятся к верхнему регистру.
func paramsFromProto(in *antifraudv1.RuleParams) (*models.RuleParams, error) {
	if in == nil {
		return nil, nil
	}

	params := &models.RuleParams{
		MaxCount:      int(in.MaxCount),
		WindowSeconds: int(in.WindowSeconds),
	}

	if len(in.Limits) > 0 {
		params.Limits = make(map[models.Currency]int64, len(in.Limits))
	}

	for raw, limit := range in.Limits {
		currency, err := models.ParseCurrency(raw)
		if err != nil {
			return nil, err
		}

		params.Limits[currency] = limit
	}

	return params, nil
}
