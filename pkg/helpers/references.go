/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package helpers

import (
	"strings"

	"github.com/asaskevich/govalidator"
)

func StripRegistryFromImage(image string) string {
	img := strings.SplitN(image, "/", 2)
	if len(img) == 2 && govalidator.IsURL(img[0]) {
		return img[1]
	}
	return image
}
