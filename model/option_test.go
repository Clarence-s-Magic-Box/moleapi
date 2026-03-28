package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapKeepsDefaultImageOutputRatioWhenStoredValueIsBlank(t *testing.T) {
	originalOptionMap := common.OptionMap
	originalImageOutputJSON := ratio_setting.ImageOutputRatio2JSONString()

	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
		require.NoError(t, ratio_setting.UpdateImageOutputRatioByJSONString(originalImageOutputJSON))
	})

	require.NoError(t, ratio_setting.UpdateImageOutputRatioByJSONString(`{"gemini-3.1-flash-image-preview":60}`))
	common.OptionMap = map[string]string{
		"ImageOutputRatio": ratio_setting.ImageOutputRatio2JSONString(),
	}

	err := updateOptionMap("ImageOutputRatio", "")
	require.NoError(t, err)

	require.Equal(t, `{"gemini-3.1-flash-image-preview":60}`, common.OptionMap["ImageOutputRatio"])
	ratio, ok := ratio_setting.GetImageOutputRatio("gemini-3.1-flash-image-preview")
	require.True(t, ok)
	require.Equal(t, 60.0, ratio)
}
