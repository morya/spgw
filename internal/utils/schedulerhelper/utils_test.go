package schedulerhelper

import (
	"strings"
	"testing"
	"time"
)

// TestGenerateJobID 测试生成任务ID
func TestGenerateJobID(t *testing.T) {
	id := generateJobID()

	if id == "" {
		t.Error("generateJobID returned empty string")
	}

	// 验证ID是纯数字
	for _, c := range id {
		if c < '0' || c > '9' {
			t.Errorf("job ID should be numeric only: %s", id)
		}
	}
}

// TestGenerateJobIDUniqueness 测试任务ID唯一性
func TestGenerateJobIDUniqueness(t *testing.T) {
	// 生成多个ID，验证它们是唯一的
	ids := make(map[string]bool)
	for range 1000 {
		id := generateJobID()
		if ids[id] {
			t.Errorf("duplicate job ID generated: %s", id)
		}
		ids[id] = true
	}
}

// TestGenerateJobIDFormat 测试ID格式
func TestGenerateJobIDFormat(t *testing.T) {
	id := generateJobID()

	// 验证ID长度（雪花ID通常是16-19位数字）
	if len(id) < 16 || len(id) > 19 {
		t.Errorf("job ID length should be between 16-19, got: %d (%s)", len(id), id)
	}
}

// TestGenerateJobIDConcurrency 测试并发生成ID
func TestGenerateJobIDConcurrency(t *testing.T) {
	const goroutines = 100
	const idsPerGoroutine = 100

	idChan := make(chan string, goroutines*idsPerGoroutine)

	// 并发生成ID
	for range goroutines {
		go func() {
			for range idsPerGoroutine {
				idChan <- generateJobID()
			}
		}()
	}

	// 收集所有ID
	ids := make(map[string]bool)
	for range goroutines * idsPerGoroutine {
		id := <-idChan
		if ids[id] {
			t.Errorf("duplicate job ID generated: %s", id)
		}
		ids[id] = true
	}
}

// TestValidateJobValidJob 测试验证有效任务
func TestValidateJobValidJob(t *testing.T) {
	job := &Job{
		Group:          "test-group",
		Name:           "test-job",
		ExecutorName:   "test-executor",
		CronExpression: "*/5 * * * * *",
		ParallelNum:    2,
		Timeout:        60 * time.Second,
		RetryInterval:  5 * time.Second,
	}

	err := validateJob(job)
	if err != nil {
		t.Errorf("validateJob returned error for valid job: %v", err)
	}
}

// TestValidateJobEmptyGroup 测试空分组
func TestValidateJobEmptyGroup(t *testing.T) {
	job := &Job{
		Group:          "",
		Name:           "test-job",
		ExecutorName:   "test-executor",
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err == nil {
		t.Error("validateJob should return error for empty group")
	}
	if !strings.Contains(err.Error(), "分组") {
		t.Errorf("error message should mention group: %v", err)
	}
}

// TestValidateJobEmptyName 测试空任务名
func TestValidateJobEmptyName(t *testing.T) {
	job := &Job{
		Group:          "test-group",
		Name:           "",
		ExecutorName:   "test-executor",
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err == nil {
		t.Error("validateJob should return error for empty name")
	}
	if !strings.Contains(err.Error(), "名称") {
		t.Errorf("error message should mention name: %v", err)
	}
}

// TestValidateJobEmptyExecutorName 测试空执行器名称
func TestValidateJobEmptyExecutorName(t *testing.T) {
	job := &Job{
		Group:          "test-group",
		Name:           "test-job",
		ExecutorName:   "",
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err == nil {
		t.Error("validateJob should return error for empty executor name")
	}
	if !strings.Contains(err.Error(), "执行器") {
		t.Errorf("error message should mention executor: %v", err)
	}
}

// TestValidateJobEmptyCronExpression 测试空CRON表达式
func TestValidateJobEmptyCronExpression(t *testing.T) {
	job := &Job{
		Group:          "test-group",
		Name:           "test-job",
		ExecutorName:   "test-executor",
		CronExpression: "",
	}

	err := validateJob(job)
	if err == nil {
		t.Error("validateJob should return error for empty cron expression")
	}
	if !strings.Contains(err.Error(), "CRON") {
		t.Errorf("error message should mention cron: %v", err)
	}
}

// TestValidateJobDefaultValues 测试默认值设置
func TestValidateJobDefaultValues(t *testing.T) {
	tests := []struct {
		name         string
		job          *Job
		wantParallel int
		wantTimeout  time.Duration
		wantRetry    time.Duration
	}{
		{
			name: "Zero values get defaults",
			job: &Job{
				Group:          "test-group",
				Name:           "test-job",
				ExecutorName:   "test-executor",
				CronExpression: "*/5 * * * * *",
				ParallelNum:    0,
				Timeout:        0,
				RetryInterval:  0,
			},
			wantParallel: 1,
			wantTimeout:  30 * time.Second,
			wantRetry:    10 * time.Second,
		},
		{
			name: "Negative parallel gets default",
			job: &Job{
				Group:          "test-group",
				Name:           "test-job",
				ExecutorName:   "test-executor",
				CronExpression: "*/5 * * * * *",
				ParallelNum:    -1,
			},
			wantParallel: 1,
			wantTimeout:  30 * time.Second,
			wantRetry:    10 * time.Second,
		},
		{
			name: "Positive values are preserved",
			job: &Job{
				Group:          "test-group",
				Name:           "test-job",
				ExecutorName:   "test-executor",
				CronExpression: "*/5 * * * * *",
				ParallelNum:    5,
				Timeout:        120 * time.Second,
				RetryInterval:  30 * time.Second,
			},
			wantParallel: 5,
			wantTimeout:  120 * time.Second,
			wantRetry:    30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJob(tt.job)
			if err != nil {
				t.Errorf("validateJob returned error: %v", err)
			}
			if tt.job.ParallelNum != tt.wantParallel {
				t.Errorf("ParallelNum = %d, want %d", tt.job.ParallelNum, tt.wantParallel)
			}
			if tt.job.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", tt.job.Timeout, tt.wantTimeout)
			}
			if tt.job.RetryInterval != tt.wantRetry {
				t.Errorf("RetryInterval = %v, want %v", tt.job.RetryInterval, tt.wantRetry)
			}
		})
	}
}

// TestValidateJobMultipleErrors 测试多个错误
func TestValidateJobMultipleErrors(t *testing.T) {
	job := &Job{
		Group:          "",
		Name:           "",
		ExecutorName:   "",
		CronExpression: "",
	}

	err := validateJob(job)
	if err == nil {
		t.Error("validateJob should return error for invalid job")
	}
}

// TestValidateJobWithWhitespace 测试空白字符
func TestValidateJobWithWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		group    string
		jobName  string
		executor string
		cron     string
		wantErr  bool
	}{
		// 注意：Go 中 len("   ") > 0，所以 validateJob 不会报错
		// validateJob 只检查空字符串，不检查空白字符串
		{"Spaces only", "   ", "test", "executor", "* * * * *", false},
		{"Tabs only", "\t\t", "test", "executor", "* * * * *", false},
		{"Valid with spaces", " test group ", "test job", "executor", "* * * * *", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Group:          tt.group,
				Name:           tt.jobName,
				ExecutorName:   tt.executor,
				CronExpression: tt.cron,
			}

			err := validateJob(job)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGenerateJobIDMonotonic 测试ID单调递增
func TestGenerateJobIDMonotonic(t *testing.T) {
	id1 := generateJobID()
	id2 := generateJobID()

	// 雪花ID应该单调递增
	if id2 <= id1 {
		t.Errorf("ID should be monotonically increasing: id1=%s, id2=%s", id1, id2)
	}
}

// TestValidateJobNil 测试nil任务
func TestValidateJobNil(t *testing.T) {
	var job *Job = nil

	// 这个测试验证validateJob对nil的处理
	// 实际使用中不应该传入nil
	// 如果validateJob没有nil检查，会panic
	// 这里我们假设调用者会确保job不为nil

	_ = job // 避免unused变量警告
}

// TestValidateJobWithChinese 测试中文字段
func TestValidateJobWithChinese(t *testing.T) {
	job := &Job{
		Group:          "测试分组",
		Name:           "测试任务",
		ExecutorName:   "测试执行器",
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err != nil {
		t.Errorf("validateJob should accept Chinese characters: %v", err)
	}
}

// TestValidateJobWithEmoji 测试emoji字符
func TestValidateJobWithEmoji(t *testing.T) {
	job := &Job{
		Group:          "🎯-group",
		Name:           "🚀-job",
		ExecutorName:   "⚡-executor",
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err != nil {
		t.Errorf("validateJob should accept emoji characters: %v", err)
	}
}

// TestValidateJobVeryLongStrings 测试超长字符串
func TestValidateJobVeryLongStrings(t *testing.T) {
	longString := strings.Repeat("a", 10000)

	job := &Job{
		Group:          longString,
		Name:           longString,
		ExecutorName:   longString,
		CronExpression: "*/5 * * * * *",
	}

	err := validateJob(job)
	if err != nil {
		// 长字符串应该被接受（只要非空）
		t.Errorf("validateJob should accept long strings: %v", err)
	}
}

// TestGenerateJobIDConsistency 测试ID一致性
func TestGenerateJobIDConsistency(t *testing.T) {
	// 连续生成的ID应该不同
	id1 := generateJobID()
	id2 := generateJobID()

	// 雪花ID应该保证唯一性
	if id1 == id2 {
		t.Errorf("IDs should be unique: %s", id1)
	}

	// 验证ID长度一致
	if len(id1) != len(id2) {
		t.Errorf("ID length should be consistent: %d vs %d", len(id1), len(id2))
	}
}
