package pulse

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
)

type surveyComment struct {
	ID     string `json:"id"`
	Body   string `json:"body"`
	Status string `json:"status"`
}

func writeCommentsXLSX(w http.ResponseWriter, rows []surveyComment) {
	const ns = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	var buffer bytes.Buffer
	book := zip.NewWriter(&buffer)
	files := []struct{ path, contents string }{
		{"[Content_Types].xml", `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`},
		{"_rels/.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<workbook xmlns="` + ns + `" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="التعليقات" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
	}
	for _, file := range files {
		entry, err := book.Create(file.path)
		if err == nil {
			_, err = io.WriteString(entry, file.contents)
		}
		if err != nil {
			fail(w, 500, "export_error")
			return
		}
	}
	sheet, err := book.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		fail(w, 500, "export_error")
		return
	}
	io.WriteString(sheet, `<worksheet xmlns="`+ns+`"><sheetViews><sheetView workbookViewId="0" rightToLeft="1"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols><col min="1" max="1" width="95" customWidth="1"/><col min="2" max="2" width="22" customWidth="1"/></cols><sheetData>`)
	writeRow := func(body, status string) {
		io.WriteString(sheet, "<row>")
		for _, value := range []string{body, status} {
			io.WriteString(sheet, `<c t="inlineStr"><is><t xml:space="preserve">`)
			xml.EscapeText(sheet, []byte(value))
			io.WriteString(sheet, "</t></is></c>")
		}
		io.WriteString(sheet, "</row>")
	}
	writeRow("التعليق", "حالة المراجعة")
	labels := map[string]string{"approved": "معتمد", "rejected": "مستبعد", "pending": "بانتظار المراجعة"}
	for _, comment := range rows {
		writeRow(comment.Body, labels[comment.Status])
	}
	io.WriteString(sheet, "</sheetData></worksheet>")
	if err := book.Close(); err != nil {
		fail(w, 500, "export_error")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="Deraah_Pulse_Comments.xlsx"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(buffer.Bytes())
}
