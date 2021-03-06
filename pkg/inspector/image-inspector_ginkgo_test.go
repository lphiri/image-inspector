// +build integrationtest

package inspector_test

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	iicmd "github.com/openshift/image-inspector/pkg/cmd"
	"github.com/openshift/image-inspector/pkg/imageserver"
	. "github.com/openshift/image-inspector/pkg/inspector"
)

var _ = Describe("ImageInspector", func() {
	var (
		ii           ImageInspector
		opts         *iicmd.ImageInspectorOptions
		serve        = "localhost:8088"
		validToken   = "w599voG89897rGVDmdp12WA681r9E5948c1CJTPi8g4HGc4NWaz62k6k1K0FMxHW40H8yOO3Hoe"
		invalidToken = "asdfqwer1234"
		client       = http.Client{
			Timeout: time.Minute,
		}
	)
	JustBeforeEach(func() {
		opts = iicmd.NewDefaultImageInspectorOptions()
		opts.Serve = serve
		opts.AuthToken = validToken
		opts.Image = "fedora:22"
		opts.ScanType = "openscap"

		ii = NewDefaultImageInspector(*opts)

	})
	Describe(".Inspect()", func() {
		//note: no expects in this block
		//we just begin the http server here
		It("starts running witouth error", func() {
			//serving blocks, so it needs to be done in a goroutine
			go func() {
				if err := ii.Inspect(); err != nil {
					panic(err)
				}
			}()
			//allow 3 minutes to pull image
			if err := waitForImage(opts.URI, opts.Image, time.Minute*3); err != nil {
				panic(err)
			}
			//allow 30s to start serving http
			if err := waitForServer(opts.Serve, time.Second*30); err != nil {
				panic(err)
			}
		})

		paths := []string{
			//HEALTHZ_URL_PATH,
			//API_URL_PREFIX,
			//METADATA_URL_PATH,
			//OPENSCAP_URL_PATH,
			//OPENSCAP_REPORT_URL_PATH,
			CONTENT_URL_PREFIX,
		}
		for _, path := range paths {
			Context("when user sends HTTP request to "+path, func() {
				var req *http.Request
				BeforeEach(func() {
					var err error
					req, err = http.NewRequest("GET", "http://"+serve+path, nil)
					if err != nil {
						panic(err)
					}
				})
				Context("with incorrect authentication token", func() {
					BeforeEach(func() {
						req.Header.Set(imageserver.AUTH_TOKEN_HEADER, invalidToken)
					})
					It("should fail with status http.Status BadRequest", func() {
						res, err := client.Do(req)
						Expect(err).NotTo(HaveOccurred())
						Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
					})
				})
				Context("with correct authentication token", func() {
					BeforeEach(func() {
						req.Header.Set(imageserver.AUTH_TOKEN_HEADER, validToken)
					})
					It("should authorize the request", func() {
						res, err := client.Do(req)
						Expect(err).NotTo(HaveOccurred())
						Expect(res.StatusCode).NotTo(Equal(http.StatusUnauthorized))
					})
				})
			})
		}
	})
})

func waitForImage(uri, imageName string, timeout time.Duration) error {
	client, err := docker.NewClient(uri)
	if err != nil {
		return err
	}
	errchan := make(chan error, 1)
	pollFunc := func() {
		images, err := client.ListImages(docker.ListImagesOptions{})
		if err != nil {
			errchan <- err
			return
		}
		for _, image := range images {
			for _, tag := range image.RepoTags {
				if strings.Contains(tag, imageName) {
					errchan <- nil
					return
				}
			}
		}
	}
	go func() {
		for {
			pollFunc()
			time.Sleep(time.Second * 5)
		}
	}()
	select {
	case <-time.After(timeout):
		return fmt.Errorf("waiting for image timed out after %s", timeout.String())
	case err := <-errchan:
		if err != nil {
			return err
		}
		return nil
	}
}
func waitForServer(addr string, timeout time.Duration) error {
	errchan := make(chan error, 1)
	pollFunc := func() {
		req, err := http.NewRequest("GET", "http://"+addr+"/", nil)
		if err != nil {
			panic(err)
		}
		if _, err := http.DefaultClient.Do(req); err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				return //still waiting
			}
			errchan <- err
			return
		}
		errchan <- nil
		return
	}
	go func() {
		for {
			pollFunc()
			time.Sleep(time.Second * 5)
		}
	}()
	select {
	case <-time.After(timeout):
		return fmt.Errorf("waiting for sever timed out after %s", timeout.String())
	case err := <-errchan:
		if err != nil {
			return err
		}
		return nil
	}
}
