package cbr

import (
	"testing"
)

func TestParseKeyRatePercent_picksLatestByDate(t *testing.T) {
	t.Parallel()
	// Так ЦБ обычно отдаёт: свежие даты сверху.
	const xml = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <KeyRateResponse xmlns="http://web.cbr.ru/">
      <KeyRateResult>
        <diffgr:diffgram xmlns:diffgr="urn:schemas-microsoft-com:xml-diffgram-v1">
          <KeyRate xmlns="">
            <KR><DT>2025-05-07T00:00:00+03:00</DT><Rate>21.00</Rate></KR>
            <KR><DT>2025-04-01T00:00:00+03:00</DT><Rate>16.00</Rate></KR>
          </KeyRate>
        </diffgr:diffgram>
      </KeyRateResult>
    </KeyRateResponse>
  </soap:Body>
</soap:Envelope>`
	got, err := ParseKeyRatePercent([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got != 21.0 {
		t.Fatalf("got %v, want 21 - должна взяться ставка на самую позднюю дату", got)
	}
}

func TestParseKeyRatePercent_reverseOrderStillLatest(t *testing.T) {
	t.Parallel()
	const xml = `<?xml version="1.0" encoding="utf-8"?>
<Envelope><diffgram><KeyRate xmlns="">
  <KR><DT>2025-04-01T00:00:00+03:00</DT><Rate>16.00</Rate></KR>
  <KR><DT>2025-05-07T00:00:00+03:00</DT><Rate>21.00</Rate></KR>
</KeyRate></diffgram></Envelope>`
	got, err := ParseKeyRatePercent([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got != 21.0 {
		t.Fatalf("got %v want 21", got)
	}
}
