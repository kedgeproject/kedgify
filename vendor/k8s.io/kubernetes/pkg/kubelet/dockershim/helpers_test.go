/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dockershim

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	dockertypes "github.com/docker/engine-api/types"
	dockernat "github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/kubernetes/pkg/api/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/runtime"
	"k8s.io/kubernetes/pkg/kubelet/dockertools"
	"k8s.io/kubernetes/pkg/security/apparmor"
)

func TestLabelsAndAnnotationsRoundTrip(t *testing.T) {
	expectedLabels := map[string]string{"foo.123.abc": "baz", "bar.456.xyz": "qwe"}
	expectedAnnotations := map[string]string{"uio.ert": "dfs", "jkl": "asd"}
	// Merge labels and annotations into docker labels.
	dockerLabels := makeLabels(expectedLabels, expectedAnnotations)
	// Extract labels and annotations from docker labels.
	actualLabels, actualAnnotations := extractLabels(dockerLabels)
	assert.Equal(t, expectedLabels, actualLabels)
	assert.Equal(t, expectedAnnotations, actualAnnotations)
}

// TestGetContainerSecurityOpts tests the logic of generating container security options from sandbox annotations.
// The actual profile loading logic is tested in dockertools.
// TODO: Migrate the corresponding test to dockershim.
func TestGetContainerSecurityOpts(t *testing.T) {
	containerName := "bar"
	makeConfig := func(annotations map[string]string) *runtimeapi.PodSandboxConfig {
		return makeSandboxConfigWithLabelsAndAnnotations("pod", "ns", "1234", 1, nil, annotations)
	}

	tests := []struct {
		msg          string
		config       *runtimeapi.PodSandboxConfig
		expectedOpts []string
	}{{
		msg:          "No security annotations",
		config:       makeConfig(nil),
		expectedOpts: []string{"seccomp=unconfined"},
	}, {
		msg: "Seccomp unconfined",
		config: makeConfig(map[string]string{
			v1.SeccompContainerAnnotationKeyPrefix + containerName: "unconfined",
		}),
		expectedOpts: []string{"seccomp=unconfined"},
	}, {
		msg: "Seccomp default",
		config: makeConfig(map[string]string{
			v1.SeccompContainerAnnotationKeyPrefix + containerName: "docker/default",
		}),
		expectedOpts: nil,
	}, {
		msg: "Seccomp pod default",
		config: makeConfig(map[string]string{
			v1.SeccompPodAnnotationKey: "docker/default",
		}),
		expectedOpts: nil,
	}, {
		msg: "AppArmor runtime/default",
		config: makeConfig(map[string]string{
			apparmor.ContainerAnnotationKeyPrefix + containerName: apparmor.ProfileRuntimeDefault,
		}),
		expectedOpts: []string{"seccomp=unconfined"},
	}, {
		msg: "AppArmor local profile",
		config: makeConfig(map[string]string{
			apparmor.ContainerAnnotationKeyPrefix + containerName: apparmor.ProfileNamePrefix + "foo",
		}),
		expectedOpts: []string{"seccomp=unconfined", "apparmor=foo"},
	}, {
		msg: "AppArmor and seccomp profile",
		config: makeConfig(map[string]string{
			v1.SeccompContainerAnnotationKeyPrefix + containerName: "docker/default",
			apparmor.ContainerAnnotationKeyPrefix + containerName:  apparmor.ProfileNamePrefix + "foo",
		}),
		expectedOpts: []string{"apparmor=foo"},
	}}

	for i, test := range tests {
		opts, err := getContainerSecurityOpts(containerName, test.config, "test/seccomp/profile/root", '=')
		assert.NoError(t, err, "TestCase[%d]: %s", i, test.msg)
		assert.Len(t, opts, len(test.expectedOpts), "TestCase[%d]: %s", i, test.msg)
		for _, opt := range test.expectedOpts {
			assert.Contains(t, opts, opt, "TestCase[%d]: %s", i, test.msg)
		}
	}
}

// TestGetSandboxSecurityOpts tests the logic of generating sandbox security options from sandbox annotations.
func TestGetSandboxSecurityOpts(t *testing.T) {
	makeConfig := func(annotations map[string]string) *runtimeapi.PodSandboxConfig {
		return makeSandboxConfigWithLabelsAndAnnotations("pod", "ns", "1234", 1, nil, annotations)
	}

	tests := []struct {
		msg          string
		config       *runtimeapi.PodSandboxConfig
		expectedOpts []string
	}{{
		msg:          "No security annotations",
		config:       makeConfig(nil),
		expectedOpts: []string{"seccomp=unconfined"},
	}, {
		msg: "Seccomp default",
		config: makeConfig(map[string]string{
			v1.SeccompPodAnnotationKey: "docker/default",
		}),
		expectedOpts: nil,
	}, {
		msg: "Seccomp unconfined",
		config: makeConfig(map[string]string{
			v1.SeccompPodAnnotationKey: "unconfined",
		}),
		expectedOpts: []string{"seccomp=unconfined"},
	}, {
		msg: "Seccomp pod and container profile",
		config: makeConfig(map[string]string{
			v1.SeccompContainerAnnotationKeyPrefix + "test-container": "unconfined",
			v1.SeccompPodAnnotationKey:                                "docker/default",
		}),
		expectedOpts: nil,
	}}

	for i, test := range tests {
		opts, err := getSandboxSecurityOpts(test.config, "test/seccomp/profile/root", '=')
		assert.NoError(t, err, "TestCase[%d]: %s", i, test.msg)
		assert.Len(t, opts, len(test.expectedOpts), "TestCase[%d]: %s", i, test.msg)
		for _, opt := range test.expectedOpts {
			assert.Contains(t, opts, opt, "TestCase[%d]: %s", i, test.msg)
		}
	}
}

// TestGetSystclsFromAnnotations tests the logic of getting sysctls from annotations.
func TestGetSystclsFromAnnotations(t *testing.T) {
	tests := []struct {
		annotations     map[string]string
		expectedSysctls map[string]string
	}{{
		annotations: map[string]string{
			v1.SysctlsPodAnnotationKey:       "kernel.shmmni=32768,kernel.shmmax=1000000000",
			v1.UnsafeSysctlsPodAnnotationKey: "knet.ipv4.route.min_pmtu=1000",
		},
		expectedSysctls: map[string]string{
			"kernel.shmmni":            "32768",
			"kernel.shmmax":            "1000000000",
			"knet.ipv4.route.min_pmtu": "1000",
		},
	}, {
		annotations: map[string]string{
			v1.SysctlsPodAnnotationKey: "kernel.shmmni=32768,kernel.shmmax=1000000000",
		},
		expectedSysctls: map[string]string{
			"kernel.shmmni": "32768",
			"kernel.shmmax": "1000000000",
		},
	}, {
		annotations: map[string]string{
			v1.UnsafeSysctlsPodAnnotationKey: "knet.ipv4.route.min_pmtu=1000",
		},
		expectedSysctls: map[string]string{
			"knet.ipv4.route.min_pmtu": "1000",
		},
	}}

	for i, test := range tests {
		actual, err := getSysctlsFromAnnotations(test.annotations)
		assert.NoError(t, err, "TestCase[%d]", i)
		assert.Len(t, actual, len(test.expectedSysctls), "TestCase[%d]", i)
		assert.Equal(t, test.expectedSysctls, actual, "TestCase[%d]", i)
	}
}

// TestGetUserFromImageUser tests the logic of getting image uid or user name of image user.
func TestGetUserFromImageUser(t *testing.T) {
	newI64 := func(i int64) *int64 { return &i }
	for c, test := range map[string]struct {
		user string
		uid  *int64
		name string
	}{
		"no gid": {
			user: "0",
			uid:  newI64(0),
		},
		"uid/gid": {
			user: "0:1",
			uid:  newI64(0),
		},
		"empty user": {
			user: "",
		},
		"multiple spearators": {
			user: "1:2:3",
			uid:  newI64(1),
		},
		"root username": {
			user: "root:root",
			name: "root",
		},
		"username": {
			user: "test:test",
			name: "test",
		},
	} {
		t.Logf("TestCase - %q", c)
		actualUID, actualName := getUserFromImageUser(test.user)
		assert.Equal(t, test.uid, actualUID)
		assert.Equal(t, test.name, actualName)
	}
}

func TestParsingCreationConflictError(t *testing.T) {
	// Expected error message from docker.
	msg := "Conflict. The name \"/k8s_POD_pfpod_e2e-tests-port-forwarding-dlxt2_81a3469e-99e1-11e6-89f2-42010af00002_0\" is already in use by container 24666ab8c814d16f986449e504ea0159468ddf8da01897144a770f66dce0e14e. You have to remove (or rename) that container to be able to reuse that name."

	matches := conflictRE.FindStringSubmatch(msg)
	require.Len(t, matches, 2)
	require.Equal(t, matches[1], "24666ab8c814d16f986449e504ea0159468ddf8da01897144a770f66dce0e14e")
}

func TestGetSecurityOptSeparator(t *testing.T) {
	for c, test := range map[string]struct {
		desc     string
		version  *semver.Version
		expected rune
	}{
		"older docker version": {
			version:  &semver.Version{Major: 1, Minor: 22, Patch: 0},
			expected: ':',
		},
		"changed docker version": {
			version:  &semver.Version{Major: 1, Minor: 23, Patch: 0},
			expected: '=',
		},
		"newer docker version": {
			version:  &semver.Version{Major: 1, Minor: 24, Patch: 0},
			expected: '=',
		},
	} {
		actual := getSecurityOptSeparator(test.version)
		assert.Equal(t, test.expected, actual, c)
	}
}

// writeDockerConfig will write a config file into a temporary dir, and return that dir.
// Caller is responsible for deleting the dir and its contents.
func writeDockerConfig(cfg string) (string, error) {
	tmpdir, err := ioutil.TempDir("", "dockershim=helpers_test.go=")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(tmpdir, ".docker")
	if err := os.Mkdir(dir, 0755); err != nil {
		return "", err
	}
	return tmpdir, ioutil.WriteFile(filepath.Join(dir, "config.json"), []byte(cfg), 0644)
}

func TestEnsureSandboxImageExists(t *testing.T) {
	sandboxImage := "gcr.io/test/image"
	registryHost := "https://gcr.io/"
	authConfig := dockertypes.AuthConfig{Username: "user", Password: "pass"}
	authB64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", authConfig.Username, authConfig.Password)))
	authJSON := fmt.Sprintf("{\"auths\": {\"%s\": {\"auth\": \"%s\"} } }", registryHost, authB64)
	for desc, test := range map[string]struct {
		injectImage  bool
		imgNeedsAuth bool
		injectErr    error
		calls        []string
		err          bool
		configJSON   string
	}{
		"should not pull image when it already exists": {
			injectImage: true,
			injectErr:   nil,
			calls:       []string{"inspect_image"},
		},
		"should pull image when it doesn't exist": {
			injectImage: false,
			injectErr:   dockertools.ImageNotFoundError{ID: "image_id"},
			calls:       []string{"inspect_image", "pull"},
		},
		"should return error when inspect image fails": {
			injectImage: false,
			injectErr:   fmt.Errorf("arbitrary error"),
			calls:       []string{"inspect_image"},
			err:         true,
		},
		"should return error when image pull needs private auth, but none provided": {
			injectImage:  true,
			imgNeedsAuth: true,
			injectErr:    dockertools.ImageNotFoundError{ID: "image_id"},
			calls:        []string{"inspect_image", "pull"},
			err:          true,
		},
		"should pull private image using dockerauth if image doesn't exist": {
			injectImage:  true,
			imgNeedsAuth: true,
			injectErr:    dockertools.ImageNotFoundError{ID: "image_id"},
			calls:        []string{"inspect_image", "pull"},
			configJSON:   authJSON,
			err:          false,
		},
	} {
		t.Logf("TestCase: %q", desc)
		_, fakeDocker, _ := newTestDockerService()
		if test.injectImage {
			images := []dockertypes.Image{{ID: sandboxImage}}
			fakeDocker.InjectImages(images)
			if test.imgNeedsAuth {
				fakeDocker.MakeImagesPrivate(images, authConfig)
			}
		}
		fakeDocker.InjectError("inspect_image", test.injectErr)

		var dockerCfgSearchPath []string
		if test.configJSON != "" {
			tmpdir, err := writeDockerConfig(test.configJSON)
			require.NoError(t, err, "could not create a temp docker config file")
			dockerCfgSearchPath = append(dockerCfgSearchPath, filepath.Join(tmpdir, ".docker"))
			defer os.RemoveAll(tmpdir)
		}

		err := ensureSandboxImageExistsDockerCfg(fakeDocker, sandboxImage, dockerCfgSearchPath)
		assert.NoError(t, fakeDocker.AssertCalls(test.calls))
		assert.Equal(t, test.err, err != nil)
	}
}

func TestMakePortsAndBindings(t *testing.T) {
	for desc, test := range map[string]struct {
		pm           []*runtimeapi.PortMapping
		exposedPorts map[dockernat.Port]struct{}
		portmappings map[dockernat.Port][]dockernat.PortBinding
	}{
		"no port mapping": {
			pm:           nil,
			exposedPorts: map[dockernat.Port]struct{}{},
			portmappings: map[dockernat.Port][]dockernat.PortBinding{},
		},
		"tcp port mapping": {
			pm: []*runtimeapi.PortMapping{
				{
					Protocol:      runtimeapi.Protocol_TCP,
					ContainerPort: 80,
					HostPort:      80,
				},
			},
			exposedPorts: map[dockernat.Port]struct{}{
				"80/tcp": {},
			},
			portmappings: map[dockernat.Port][]dockernat.PortBinding{
				"80/tcp": {
					{
						HostPort: "80",
					},
				},
			},
		},
		"udp port mapping": {
			pm: []*runtimeapi.PortMapping{
				{
					Protocol:      runtimeapi.Protocol_UDP,
					ContainerPort: 80,
					HostPort:      80,
				},
			},
			exposedPorts: map[dockernat.Port]struct{}{
				"80/udp": {},
			},
			portmappings: map[dockernat.Port][]dockernat.PortBinding{
				"80/udp": {
					{
						HostPort: "80",
					},
				},
			},
		},
		"multipe port mappings": {
			pm: []*runtimeapi.PortMapping{
				{
					Protocol:      runtimeapi.Protocol_TCP,
					ContainerPort: 80,
					HostPort:      80,
				},
				{
					Protocol:      runtimeapi.Protocol_TCP,
					ContainerPort: 80,
					HostPort:      81,
				},
			},
			exposedPorts: map[dockernat.Port]struct{}{
				"80/tcp": {},
			},
			portmappings: map[dockernat.Port][]dockernat.PortBinding{
				"80/tcp": {
					{
						HostPort: "80",
					},
					{
						HostPort: "81",
					},
				},
			},
		},
	} {
		t.Logf("TestCase: %s", desc)
		actualExposedPorts, actualPortMappings := makePortsAndBindings(test.pm)
		assert.Equal(t, test.exposedPorts, actualExposedPorts)
		assert.Equal(t, test.portmappings, actualPortMappings)
	}
}
