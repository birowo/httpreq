package main

import (
	"github.com/birowo/httpreq"
	"github.com/panjf2000/gnet/v2"
)

// httpServer mengimplementasikan gnet.EventHandler
type httpServer struct {
	gnet.BuiltinEventEngine
}

var (
	badReqRes = []byte(
		"HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n",
	)
	pongRes = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\npong\n")
)

func (hs *httpServer) OnTraffic(c gnet.Conn) gnet.Action {
	// 1. Peek seluruh buffer tanpa alokasi memori baru
	buf, _ := c.Peek(c.InboundBuffered())

	// 2. Siapkan struct Request di stack (zero-alloc)
	req, consumed, incomplete, err := httpreq.Parse(buf, 1000000)

	// 3. Handle error format HTTP (Bad Request)
	if err != nil {
		println("error")
		c.Write(badReqRes)
		return gnet.Close
	}

	// 4. Jika paket belum lengkap, tunggu data berikutnya di event OnTraffic selanjutnya
	if incomplete {
		println("incomplete")
		return gnet.None
	}

	// --- LOGIKA BISNIS (Eksekusi sebelum c.Discard) ---

	// Kirim response balik ke client
	c.Write(pongRes)

	println()
	println("sebelum di-parse:\n", string(buf[:consumed]))
	println()
	println("setelah di-parse:")
	mthd := req.Method
	path := req.Path
	println("method:", string(buf[mthd.Bgn:mthd.End]), "\npath:", string(buf[path.Bgn:path.End]), "\nquery:", string(req.Query), "\nproto:", string(req.Proto))
	for _, hdr := range req.Headers[:req.HdrsNum] {
		println(string(hdr.Key), ":", string(hdr.Val))
	}
	if len(req.Body) != 0 {
		println("body:", string(req.Body))
	}

	// 5. Geser/buang buffer gnet yang sudah selesai diproses
	c.Discard(consumed)

	return gnet.None
}

func main() {
	// Menjalankan gnet server langsung ke localhost:8080 dengan multicore aktif
	err := gnet.Run(
		&httpServer{},
		"tcp://localhost:8080",
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
	)
	if err != nil {
		println(
			"Gagal menjalankan gnet server:", err.Error(),
		)
	}
}
