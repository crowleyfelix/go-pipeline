package log

var (
	logger Logger
)

func Setup(l Logger) {
	logger = l
}

func init() {
	Setup(StandardLogger{})
}
