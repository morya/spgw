package billdumper

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/models"
)

func archiveSpBill(sw *SpBillFileWriter, nodeID uint32) error {
	if sw == nil {
		return fmt.Errorf("nil sp bill writer")
	}

	if sw.billCount == 0 {
		os.Remove(sw.filePath)
		return nil
	}

	gzPath := sw.filePath + ".gz"
	originalSize, err := gzipFile(sw.filePath, gzPath)
	if err != nil {
		return fmt.Errorf("gzip %s failed: %w", sw.filePath, err)
	}

	if err := os.Remove(sw.filePath); err != nil {
		glog.RunS().Warnf("remove original sp bill file %s failed: %v", sw.filePath, err)
	}

	gzInfo, err := os.Stat(gzPath)
	if err != nil {
		return fmt.Errorf("stat gz file %s failed: %w", gzPath, err)
	}

	billFile := &models.SpBillFile{
		Spid:           sw.spid,
		FileName:       filepath.Base(gzPath),
		OriginalSize:   originalSize,
		CompressedSize: gzInfo.Size(),
		BillCount:      sw.billCount,
		StartTime:      sw.startTime,
		EndTime:        sw.endTime,
		NodeID:         nodeID,
	}
	if err := db.MainDB.Create(billFile).Error; err != nil {
		return fmt.Errorf("insert spbill_file to db failed: %w", err)
	}
	return nil
}

func gzipFile(src, dst string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sf.Close()

	stat, err := sf.Stat()
	if err != nil {
		return 0, err
	}

	df, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer df.Close()

	gw := gzip.NewWriter(df)
	defer gw.Close()

	if _, err := io.Copy(gw, sf); err != nil {
		return 0, err
	}
	return stat.Size(), nil
}
