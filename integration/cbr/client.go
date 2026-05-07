package cbr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
)

func BuildSOAPRequest() string {
	fromDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <KeyRate xmlns="http://web.cbr.ru/">
      <fromDate>%s</fromDate>
      <ToDate>%s</ToDate>
    </KeyRate>
  </soap12:Body>
</soap12:Envelope>`, fromDate, toDate)
}

func SendRequest(soapRequest string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(
		http.MethodPost,
		"https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx",
		bytes.NewBufferString(soapRequest),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cbr request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cbr status %d", resp.StatusCode)
	}
	return raw, nil
}

func ParseKeyRatePercent(rawBody []byte) (float64, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawBody); err != nil {
		return 0, fmt.Errorf("parse xml: %w", err)
	}
	krElements := doc.FindElements("//diffgram/KeyRate/KR")
	if len(krElements) == 0 {
		krElements = doc.FindElements("//KR")
	}
	if len(krElements) == 0 {
		return 0, errors.New("key rate KR elements not found")
	}
	// Берём ставку по самой свежей дате (DT), а не просто последнюю строку в XML:
	// ЦБ не гарантирует порядок.
	var chosen *etree.Element
	var chosenAt time.Time
	for _, kr := range krElements {
		dtEl := kr.FindElement("./DT")
		if dtEl == nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(dtEl.Text()))
		if err != nil {
			continue
		}
		if chosen == nil || ts.After(chosenAt) {
			chosen = kr
			chosenAt = ts
		}
	}
	if chosen == nil {
		return 0, errors.New("no KR with parseable DT")
	}
	rateEl := chosen.FindElement("./Rate")
	if rateEl == nil {
		return 0, errors.New("Rate tag missing")
	}
	var rate float64
	if _, err := fmt.Sscanf(rateEl.Text(), "%f", &rate); err != nil {
		return 0, fmt.Errorf("parse rate: %w", err)
	}
	return rate, nil
}

func FetchKeyRatePercent() (float64, error) {
	raw, err := SendRequest(BuildSOAPRequest())
	if err != nil {
		return 0, err
	}
	return ParseKeyRatePercent(raw)
}
