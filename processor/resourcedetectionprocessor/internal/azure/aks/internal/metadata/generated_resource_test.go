// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceBuilder(t *testing.T) {
	for _, tt := range []string{"default", "all_set", "none_set"} {
		t.Run(tt, func(t *testing.T) {
			cfg := loadResourceAttributesConfig(t, tt)
			rb := NewResourceBuilder(cfg)
			rb.SetCloudPlatform("cloud.platform-val")
			rb.SetCloudProvider("cloud.provider-val")
			rb.SetK8sClusterName("k8s.cluster.name-val")

			res := rb.Emit()
			assert.Equal(t, 0, rb.Emit().Attributes().Len()) // Second call should return empty Resource

			switch tt {
			case "default":
				assert.Equal(t, 2, res.Attributes().Len())
			case "all_set":
				assert.Equal(t, 3, res.Attributes().Len())
			case "none_set":
				assert.Equal(t, 0, res.Attributes().Len())
				return
			default:
				assert.Failf(t, "unexpected test case: %s", tt)
			}

			val, ok := res.Attributes().Get("cloud.platform")
			assert.True(t, ok)
			if ok {
				assert.Equal(t, "cloud.platform-val", val.Str())
			}
			val, ok = res.Attributes().Get("cloud.provider")
			assert.True(t, ok)
			if ok {
				assert.Equal(t, "cloud.provider-val", val.Str())
			}
			val, ok = res.Attributes().Get("k8s.cluster.name")
			assert.Equal(t, tt == "all_set", ok)
			if ok {
				assert.Equal(t, "k8s.cluster.name-val", val.Str())
			}
		})
	}
}
