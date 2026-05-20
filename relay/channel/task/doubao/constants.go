package doubao

var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
}

var ChannelName = "doubao-video"

// videoInputRatioMap [已弃用，保留用于参考] 旧版视频输入折扣比率（仅检测 video_url 存在性）。
// 已被 resolutionVideoRatioMap 替代（四档：video_input × 分辨率）。
// var videoInputRatioMap = map[string]float64{
// 	"doubao-seedance-2-0-260128":      28.0 / 46.0, // ~0.6087
// 	"doubao-seedance-2-0-fast-260128": 22.0 / 37.0, // ~0.5946
// }

// resolutionVideoRatioMap 视频输入×分辨率四档折扣比率。
// 基准价 = 480p/720p 不含视频输入（46 元/百万tokens）。
// low=480p/720p, high=1080p; video=含视频输入, 无后缀=不含视频输入。
var resolutionVideoRatioMap = map[string]map[string]float64{
	"doubao-seedance-2-0-260128": {
		"low_video":  28.0 / 46.0, // ≈0.6087 (480p/720p + 视频)
		"high_video": 31.0 / 46.0, // ≈0.6739 (1080p + 视频)
		"high":       51.0 / 46.0, // ≈1.1087 (1080p 无视频)
	},
	"doubao-seedance-2-0-fast-260128": {
		"low_video":  22.0 / 37.0, // ≈0.5946 (480p/720p + 视频)
		"high_video": 31.0 / 46.0, // ≈0.6739 (1080p + 视频)
		"high":       51.0 / 46.0, // ≈1.1087 (1080p 无视频)
	},
}

// GetVideoInputRatio [已弃用，保留用于参考] 旧版比率查询。
// func GetVideoInputRatio(modelName string) (float64, bool) {
// 	r, ok := videoInputRatioMap[modelName]
// 	return r, ok
// }

// GetResolutionVideoRatio 根据模型名获取分辨率+视频输入四档比率映射。
func GetResolutionVideoRatio(modelName string) (map[string]float64, bool) {
	r, ok := resolutionVideoRatioMap[modelName]
	return r, ok
}
