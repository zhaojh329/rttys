/* SPDX-License-Identifier: MIT */

package server

import "testing"

func TestEmbeddedAssets(t *testing.T) {
	for _, name := range []string{"assets/dist/index.html", "assets/http-proxy-err.html"} {
		t.Run(name, func(t *testing.T) {
			if _, err := staticFs.ReadFile(name); err != nil {
				t.Fatal(err)
			}
		})
	}
}
