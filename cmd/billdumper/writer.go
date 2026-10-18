package billdumper

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/natefinch/lumberjack"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

var jsonEnc = jsoniter.ConfigCompatibleWithStandardLibrary

const alphaNum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type BillWriter struct {
	cfg          gconfig.BillDumperSetting
	nodeID       uint32
	mu           sync.Mutex
	globalWriter *lumberjack.Logger
	spWriters    map[string]*SpBillFileWriter
	spRotateAt   time.Duration
}

type SpBillFileWriter struct {
	spid      string
	nanoID    string
	file      *os.File
	filePath  string
	fileSize  int64
	billCount int64
	startTime int64
	endTime   int64
	createdAt time.Time
}

func NewBillWriter(cfg gconfig.BillDumperSetting, nodeID uint32) *BillWriter {
	return &BillWriter{
		cfg:        cfg,
		nodeID:     nodeID,
		spWriters:  map[string]*SpBillFileWriter{},
		spRotateAt: time.Duration(cfg.SpRotateIntervalSec) * time.Second,
		globalWriter: &lumberjack.Logger{
			Filename: filepath.Join(cfg.OutputPath, fmt.Sprintf("billdumper-%d-%s-%d.ndjson",
				nodeID, time.Now().Format("20060102-150405"), os.Getpid())),
			MaxSize:    cfg.MaxFileSizeMB,
			MaxBackups: orDefaultInt(cfg.MaxBackups, 30),
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
			LocalTime:  true,
		},
	}
}

func (w *BillWriter) Write(rec *spgwpb.BillRecord) error {
	data, err := jsonEnc.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.globalWriter.Write(data); err != nil {
		glog.RunS().Errorf("write global bill failed: %v", err)
	}

	spid := extractSpid(rec)
	if spid == "" {
		return nil
	}

	sw, err := w.getOrCreateSpWriter(spid)
	if err != nil {
		return err
	}

	if _, err := sw.file.Write(data); err != nil {
		return err
	}
	sw.fileSize += int64(len(data))
	sw.billCount++

	billTime := rec.BillTime
	if sw.startTime == 0 {
		sw.startTime = billTime
	}
	sw.endTime = billTime

	if sw.fileSize >= int64(w.cfg.MaxFileSizeMB)*1024*1024 ||
		time.Since(sw.createdAt) >= w.spRotateAt {
		if err := w.rotateSpWriterLocked(spid); err != nil {
			glog.RunS().Errorf("rotate sp writer failed, spid=%s, err=%v", spid, err)
		}
	}
	return nil
}

func (w *BillWriter) CheckRotate() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for spid, sw := range w.spWriters {
		if time.Since(sw.createdAt) >= w.spRotateAt {
			if err := w.rotateSpWriterLocked(spid); err != nil {
				glog.RunS().Errorf("ticker rotate sp writer failed, spid=%s, err=%v", spid, err)
			}
		}
	}
}

func (w *BillWriter) getOrCreateSpWriter(spid string) (*SpBillFileWriter, error) {
	if sw, ok := w.spWriters[spid]; ok {
		return sw, nil
	}
	sw, err := w.newSpWriter(spid)
	if err != nil {
		return nil, err
	}
	w.spWriters[spid] = sw
	return sw, nil
}

func (w *BillWriter) newSpWriter(spid string) (*SpBillFileWriter, error) {
	nanoID := gonanoid.MustGenerate(alphaNum, 16)
	fileName := fmt.Sprintf("bd-%s-%s.ndjson", spid, nanoID)
	filePath := filepath.Join(w.cfg.OutputPath, fileName)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create sp bill file %s failed: %w", filePath, err)
	}
	return &SpBillFileWriter{
		spid:      spid,
		nanoID:    nanoID,
		file:      f,
		filePath:  filePath,
		createdAt: time.Now(),
	}, nil
}

func (w *BillWriter) rotateSpWriterLocked(spid string) error {
	sw := w.spWriters[spid]
	if sw == nil {
		return nil
	}
	if err := sw.file.Close(); err != nil {
		return fmt.Errorf("close sp bill file failed: %w", err)
	}

	if err := archiveSpBill(sw, w.nodeID); err != nil {
		glog.RunS().Errorf("archive sp bill failed, spid=%s, err=%v", spid, err)
	}

	newSw, err := w.newSpWriter(spid)
	if err != nil {
		return err
	}
	w.spWriters[spid] = newSw
	return nil
}

func (w *BillWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var firstErr error
	if err := w.globalWriter.Close(); err != nil {
		firstErr = err
	}
	for spid, sw := range w.spWriters {
		if err := sw.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := archiveSpBill(sw, w.nodeID); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(w.spWriters, spid)
	}
	return firstErr
}

// extractSpid 优先 OrgSubmitMsg.Head.FromSpid（适用于 62/63 话单，OriginalMsg 是 submitAck/reportReq），
// fallback OriginalMsg.Head.FromSpid（适用于 60 话单，OriginalMsg 是 submitReq 本身）。
func extractSpid(rec *spgwpb.BillRecord) string {
	if rec == nil {
		return ""
	}
	if rec.OrgSubmitMsg != nil && rec.OrgSubmitMsg.Head != nil {
		return rec.OrgSubmitMsg.Head.FromSpid
	}
	if rec.OriginalMsg != nil && rec.OriginalMsg.Head != nil {
		return rec.OriginalMsg.Head.FromSpid
	}
	return ""
}

func orDefaultInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
