package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type WidgetResponse struct {
	Key    string `json:"key"`
	StartX int    `json:"startX"`
	StartY int    `json:"startY"`
	SizeX  int    `json:"sizeX"`
	SizeY  int    `json:"sizeY"`
}

type CreateDashboardRequest struct {
	GroupName string           `json:"groupName"`
	SizeX     int              `json:"sizeX"`
	SizeY     int              `json:"sizeY"`
	Widgets   []WidgetResponse `json:"widgets"`
}

func main() {
	input := `
		[
		  {
			"groupName": "스택정보",
			"sizeX": 4,
			"sizeY": 6,
			"widgets": [
				{"key": "PodCalendarWidget", "startX": 1, "startY": 1, "sizeX": 2, "sizeY": 2},
				{"key": "CpuUsageWidget", "startX": 1, "startY": 1, "sizeX": 2, "sizeY": 2}
			]
		  },
		  {
			"groupName": "정책정보",
			"sizeX": 4,
			"sizeY": 6,
			"widgets": [
				{"key": "PolicyViolateWidget", "startX": 1, "startY": 1, "sizeX": 2, "sizeY": 2},
				{"key": "PolicyStatusWidget", "startX": 1, "startY": 1, "sizeX": 2, "sizeY": 2}
			]
		  }
		]
    `

	var dashboard []CreateDashboardRequest
	err := Unmarshal([]byte(input), &dashboard)
	if err != nil {
		log.Fatal("error !!!!")
	}
	fmt.Printf("%+v\n\n", dashboard)

	b, err := json.Marshal(dashboard)
	if err != nil {
		log.Fatalf("Unable to unmarshal JSON due to %s", err)
	}
	content := fmt.Sprintf("%+v", string(b))
	fmt.Printf("%s\n", content)

	//err := json.Unmarshal([]byte(input), &dashboard)
	//if err != nil {
	//	log.Fatalf("Unable to marshal JSON due to %s", err)
	//}
	//
	//b, err := json.Marshal(dashboard)
	//if err != nil {
	//	log.Fatalf("Unable to unmarshal JSON due to %s", err)
	//}
	//content := fmt.Sprintf("%+v", string(b))
	//fmt.Printf("%s", content)
}

func Unmarshal(b []byte, in any) error {
	err := json.Unmarshal([]byte(b), &in)
	if err != nil {
		log.Fatalf("Unable to marshal JSON due to %s", err)
		return err
	}
	return nil
}
