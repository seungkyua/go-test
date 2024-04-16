package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type PolicyMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Kind      string `json:"kind"`
				Name      string `json:"name"`
				Violation string `json:"violation_enforcement"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PolicyTemplateMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Kind string `json:"kind"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PolicyViolationCountMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				ViolationEnforcement string `json:"violation_enforcement,omitempty"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// BarChartData Policy metric struct
type BarChartData struct {
	XAxis  *Axis        `json:"xAxis,omitempty"`
	Series []UnitNumber `json:"series,omitempty"`
}

type Axis struct {
	Data []string `json:"data"`
}

type UnitNumber struct {
	Name string `json:"name"`
	Data []int  `json:"data"`
}

type PolicyViolationMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Kind                 string `json:"kind"`
				Name                 string `json:"name"`
				Cluster              string `json:"taco_cluster"`
				ViolatingKind        string `json:"violating_kind"`
				ViolatingName        string `json:"violating_name"`
				ViolatingMsg         string `json:"violating_msg"`
				ViolationEnforcement string `json:"violation_enforcement"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type WorkloadMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type GetPolicyViolationResponse struct {
	PolicyTemplateName string
	PolicyName         string
	StackId            string
	ViolatingKind      string
	ViolatingName      string
}

func main() {
	clusters := []string{"c1", "c2", "c3"}
	clusterStr := strings.Join(clusters[:], "|")
	fmt.Printf("clusterStr ============== %s\n", clusterStr)

	query := fmt.Sprintf("sum by(kind,name,violation_enforcement)(opa_scorecard_constraint_violations{taco_cluster=~\"%s\"})", clusterStr)
	out, err := getThanosMetric(query)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}

	var pm PolicyMetric
	err = json.Unmarshal(out, &pm)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	fmt.Printf("PolicyMetric =============== %+v\n", pm)

	bcd := getBarChartData(pm)
	fmt.Printf("BarChartData =============== %+v\n", bcd)

	marshal, err := json.Marshal(bcd)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	fmt.Printf("BarChartData [json] =============== %+v\n", string(marshal))

	// ********************************************************
	// Policy Violation Log
	query = fmt.Sprintf("group(opa_scorecard_constraint_violations{taco_cluster=~\"%s\"}) "+
		"by (time, violating_kind, violating_namespace, violating_name, name, kind, violation_enforcement, violation_msg, taco_cluster)", clusterStr)
	out, err = getThanosMetric(query)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	var pvm PolicyViolationMetric
	err = json.Unmarshal(out, &pvm)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	fmt.Printf("PolicyMPolicyViolationMetric =============== %+v\n", pvm)

	// ********************************************************
	// Workload
	query = fmt.Sprintf("count (kube_deployment_status_replicas_available{taco_cluster=~'%s'} != 0)", "c3|c5")
	out, err = getThanosMetric(query)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	var wm WorkloadMetric
	err = json.Unmarshal(out, &wm)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	fmt.Printf("WorkloadMetric =============== %+v\n", wm)

	// ********************************************************
	// Policy Violation Top 5
	clusterStr = "c3"
	query = fmt.Sprintf("topk (5, sum by (kind) (opa_scorecard_constraint_violations{taco_cluster=~'%s'}))", clusterStr)
	out, err = getThanosMetric(query)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	var ptm PolicyTemplateMetric
	err = json.Unmarshal(out, &ptm)
	if err != nil {
		fmt.Printf("error - %s\n", err)
	}
	fmt.Printf("PolicyTemplateMetric =============== %+v\n", ptm)

	templateNames := make([]string, 0)
	for _, result := range ptm.Data.Result {
		templateNames = append(templateNames, result.Metric.Kind)
	}
	fmt.Printf("templateNames =============== %+v\n", templateNames)

	// X축
	var xAxis *Axis
	xData := make([]string, 0)

	// Y축
	var series []UnitNumber
	yDenyData := make([]int, 0)
	yWarnData := make([]int, 0)
	yDryrunData := make([]int, 0)

	var pvcm PolicyViolationCountMetric
	for _, templateName := range templateNames {
		xData = append(xData, templateName)

		query = fmt.Sprintf("sum by (violation_enforcement) "+
			"(opa_scorecard_constraint_violations{taco_cluster='%s', kind='%s'})", clusterStr, templateName)
		out, err = getThanosMetric(query)
		if err != nil {
			fmt.Printf("error - %s\n", err)
		}

		err = json.Unmarshal(out, &pvcm)
		if err != nil {
			fmt.Printf("error - %s\n", err)
		}
		fmt.Printf("PolicyViolationCountMetric =============== %+v\n", pvcm)

		denyCount := 0
		warnCount := 0
		dryrunCount := 0
		for _, result := range pvcm.Data.Result {
			switch policy := result.Metric.ViolationEnforcement; policy {
			case "":
				denyCount, _ = strconv.Atoi(result.Value[1].(string))
			case "warn":
				warnCount, _ = strconv.Atoi(result.Value[1].(string))
			case "dryrun":
				dryrunCount, _ = strconv.Atoi(result.Value[1].(string))
			}
		}
		yDenyData = append(yDenyData, denyCount)
		yWarnData = append(yWarnData, warnCount)
		yDryrunData = append(yDryrunData, dryrunCount)

		fmt.Printf(" =============== PolicyTemplateName: %s, deny: %d, warn: %d, dryrunCount: %d\n",
			templateName, denyCount, warnCount, dryrunCount)
	}

	xAxis = &Axis{
		Data: xData,
	}

	denyUnit := UnitNumber{
		Name: "거부",
		Data: yDenyData,
	}
	series = append(series, denyUnit)

	warnUnit := UnitNumber{
		Name: "경고",
		Data: yWarnData,
	}
	series = append(series, warnUnit)

	dryrunUnit := UnitNumber{
		Name: "감사",
		Data: yDryrunData,
	}
	series = append(series, dryrunUnit)

	bcd = &BarChartData{
		XAxis:  xAxis,
		Series: series,
	}

	fmt.Printf("PolicyViolationTop5 =============== %+v", bcd)

	bcdBytes, err := json.Marshal(bcd)
	fmt.Printf("PolicyViolationTop5 (json) =============== %+v", string(bcdBytes))

}

func getThanosMetric(query string) (out []byte, err error) {
	reqUrl := "http://siim.hopto.org:30001/api/v1/query?query=" + url.QueryEscape(query)

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{MaxIdleConns: 10},
	}

	res, err := client.Get(reqUrl)
	if err != nil {
		return out, err
	}
	if res == nil {
		return out, fmt.Errorf("failed to call thanos")
	}
	if res.StatusCode != 200 {
		return out, fmt.Errorf("invalid http status. return code: %d", res.StatusCode)
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			_ = fmt.Errorf("error closing http body")
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("body =============== %+v\n", string(body))

	return body, nil
}

func getBarChartData(pm PolicyMetric) *BarChartData {
	// totalViolation: {"K8sRequiredLabels": {"violation_enforcement": 2}}
	totalViolation := make(map[string]map[string]int)

	// Y축
	var series []UnitNumber
	var yDenyData []int
	var yWarnData []int
	var yDryrunData []int

	// X축
	var xAxis *Axis
	var xData []string

	for _, res := range pm.Data.Result {
		policyTemplate := res.Metric.Kind
		if len(res.Metric.Violation) == 0 {
			continue
		}
		fmt.Printf("policyTemplate================== %+v\n", policyTemplate)
		if !slices.Contains(xData, policyTemplate) {
			xData = append(xData, policyTemplate)
		}

		count, err := strconv.Atoi(res.Value[1].(string))
		if err != nil {
			count = 0
		}
		violation := res.Metric.Violation
		if val, ok := totalViolation[policyTemplate][violation]; !ok {
			totalViolation[policyTemplate] = make(map[string]int)
			totalViolation[policyTemplate][violation] = count
		} else {
			totalViolation[policyTemplate][violation] = val + count
		}
	}

	for _, violations := range totalViolation {
		yDenyData = append(yDenyData, violations["deny"])
		yWarnData = append(yWarnData, violations["warn"])
		yDryrunData = append(yDryrunData, violations["dryrun"])
	}

	xAxis = &Axis{
		Data: xData,
	}

	denyUnit := UnitNumber{
		Name: "거부",
		Data: yDenyData,
	}
	series = append(series, denyUnit)

	warnUnit := UnitNumber{
		Name: "경고",
		Data: yWarnData,
	}
	series = append(series, warnUnit)

	dryrunUnit := UnitNumber{
		Name: "감사",
		Data: yDryrunData,
	}
	series = append(series, dryrunUnit)

	bcd := &BarChartData{
		XAxis:  xAxis,
		Series: series,
	}

	return bcd
}
