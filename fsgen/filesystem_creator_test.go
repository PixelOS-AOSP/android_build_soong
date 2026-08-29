// Copyright 2024 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fsgen

import (
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/etc"
	"android/soong/filesystem"
	"android/soong/java"
	"android/soong/phony"

	"github.com/google/blueprint/proptools"
)

var prepareForTestWithFsgenBuildComponents = android.FixtureRegisterWithContext(registerBuildComponents)

var prepareMockRamdiksNodeList = android.FixtureMergeMockFs(android.MockFS{
	"ramdisk_node_list/ramdisk_node_list": nil,
	"ramdisk_node_list/Android.bp": []byte(`
		filegroup {
			name: "ramdisk_node_list",
			srcs: ["ramdisk_node_list"],
		}
	`),
})

var prepareForTestWithDefaultSystemDeps = android.GroupFixturePreparers(
	phony.PrepareForTestWithPhony,
	android.FixtureMergeMockFs(android.MockFS{
		"system/core/rootdir/etc/linker.config.json": nil,
		"deps/Android.bp": []byte(`
		phony {
			name: "com.android.apex.cts.shim.v1_prebuilt",
		}
		phony {
			name: "dex_bootjars",
		}
		phony {
			name: "framework_compatibility_matrix.device.xml",
		}
		phony {
			name: "init.environ.rc-soong",
		}
		phony {
			name: "libdmabufheap",
		}
		phony {
			name: "libgsi",
		}
		phony {
			name: "llndk.libraries.txt",
		}
		phony {
			name: "logpersist.start",
		}
		phony {
			name: "notice_xml_system",
		}
		phony {
			name: "system_dlkm-build.prop",
		}
		phony {
			name: "update_engine_sideload",
		}
		phony {
			name: "file_contexts_bin_gen",
		}
	`)},
	),
)

func TestFileSystemCreatorSystemImageProps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		prepareForTestWithFsgenBuildComponents,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BoardAvbEnable = true
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardAvbKeyPath:       "external/avb/test/data/testkey_rsa4096.pem",
						BoardAvbAlgorithm:     "SHA256_RSA4096",
						BoardAvbRollbackIndex: "0",
						BoardFileSystemType:   "ext4",
						BuildingImage:         true,
					},
				}
		}),
		prepareMockRamdiksNodeList,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"external/avb/test/Android.bp": []byte(`
			filegroup {
				name: "avb_testkey_rsa4096",
				srcs: ["data/testkey_rsa4096.pem"],
			}
			`),
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
		}),
	).RunTest(t)

	module := result.ModuleForTests(t, "test_product_generated_system_image", "android_common").Module()
	fooSystem := module.(interface {
		FsProps() filesystem.FilesystemProperties
	})
	android.AssertBoolEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_ENABLE'",
		true,
		proptools.Bool(fooSystem.FsProps().Use_avb),
	)
	android.AssertStringEquals(
		t,
		"Property the avb_private_key property to be set to the existing filegroup",
		":avb_testkey_rsa4096",
		proptools.String(fooSystem.FsProps().Avb_private_key),
	)
	android.AssertStringEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_ALGORITHM'",
		"SHA256_RSA4096",
		proptools.String(fooSystem.FsProps().Avb_algorithm),
	)
	android.AssertIntEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_SYSTEM_ROLLBACK_INDEX'",
		0,
		proptools.Int(fooSystem.FsProps().Rollback_index),
	)
	evaluator := module.(android.Module).ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	fsProps := fooSystem.FsProps()
	android.AssertStringEquals(
		t,
		"Property expected to match the product variable 'BOARD_SYSTEMIMAGE_FILE_SYSTEM_TYPE'",
		"ext4",
		fsProps.Type.GetOrDefault(evaluator, ""),
	)
}

func createProductPackagesSet(pkgs []string) map[string]android.ProductPackagesVariables {
	productPackagesSet := make(map[string]android.ProductPackagesVariables)
	productPackagesSet["all"] = android.ProductPackagesVariables{
		ProductPackages: pkgs,
	}
	return productPackagesSet
}

func TestFileSystemCreatorSetPartitionDeps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"bar", "baz"})
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType: "ext4",
						BuildingImage:       true,
					},
				}
		}),
		prepareMockRamdiksNodeList,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
		}),
	).RunTestWithBp(t, `
	java_library {
		name: "bar",
		srcs: ["A.java"],
	}
	java_library {
		name: "baz",
		srcs: ["A.java"],
		product_specific: true,
	}
	`)

	android.AssertBoolEquals(
		t,
		"Generated system image expected to depend on system partition installed \"bar\"",
		true,
		java.CheckModuleHasDependency(t, result.TestContext, "test_product_generated_system_image", "android_common", "bar"),
	)
	android.AssertBoolEquals(
		t,
		"Generated system image expected to not depend on product partition installed \"baz\"",
		false,
		java.CheckModuleHasDependency(t, result.TestContext, "test_product_generated_system_image", "android_common", "baz"),
	)
}

func TestFileSystemCreatorDepsWithNamespace(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		android.PrepareForTestWithNamespace,
		android.PrepareForTestWithArchMutator,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"bar"})
			config.TestProductVariables.NamespacesToExport = []string{"a/b"}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType: "ext4",
						BuildingImage:       true,
					},
				}
		}),
		android.PrepareForNativeBridgeEnabled,
		prepareMockRamdiksNodeList,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
			"a/b/Android.bp": []byte(`
			soong_namespace{
			}
			java_library {
				name: "bar",
				srcs: ["A.java"],
				compile_multilib: "64",
			}
			`),
			"c/d/Android.bp": []byte(`
			soong_namespace{
			}
			java_library {
				name: "bar",
				srcs: ["A.java"],
			}
			`),
		}),
	).RunTest(t)

	var packagingProps android.PackagingProperties
	for _, prop := range result.ModuleForTests(t, "test_product_generated_system_image", "android_common").Module().GetProperties() {
		if packagingPropStruct, ok := prop.(*android.PackagingProperties); ok {
			packagingProps = *packagingPropStruct
		}
	}
	moduleDeps := packagingProps.Multilib.Lib64.Deps

	eval := result.ModuleForTests(t, "test_product_generated_system_image", "android_common").Module().ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringListContains(
		t,
		"Generated system image expected to depend on \"bar\" defined in \"a/b\" namespace",
		moduleDeps.GetOrDefault(eval, nil),
		"//a/b:bar",
	)
	android.AssertStringListDoesNotContain(
		t,
		"Generated system image expected to not depend on \"bar\" defined in \"c/d\" namespace",
		moduleDeps.GetOrDefault(eval, nil),
		"//c/d:bar",
	)
}

func TestRemoveOverriddenModulesFromDeps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"libfoo", "libbar", "prebuiltA", "prebuiltB"})
		}),
	).RunTestWithBp(t, `
java_library {
	name: "libfoo",
}
java_library {
	name: "libbar",
	required: ["libbaz"],
}
java_library {
	name: "libbaz",
	overrides: ["libfoo"], // overrides libfoo
}
java_import {
	name: "prebuiltA",
}
java_import {
	name: "prebuiltB",
	overrides: ["prebuiltA"], // overrides prebuiltA
}
	`)
	resolvedSystemDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["system"]
	_, libFooInDeps := (*resolvedSystemDeps)["libfoo"]
	android.AssertBoolEquals(t, "libfoo should not appear in deps because it has been overridden by libbaz. The latter is a required dep of libbar, which is listed in PRODUCT_PACKAGES", false, libFooInDeps)
	_, prebuiltAInDeps := (*resolvedSystemDeps)["prebuiltA"]
	android.AssertBoolEquals(t, "prebuiltA should not appear in deps because it has been overridden by prebuiltB. The latter is listed in PRODUCT_PACKAGES", false, prebuiltAInDeps)
}

func getModuleProp[T string | bool](m android.Module, matcher func(actual interface{}) T) T {
	var defaultVal T
	for _, prop := range m.GetProperties() {
		if str := matcher(prop); str != defaultVal {
			return str
		}
	}
	return defaultVal
}

func TestPrebuiltEtcModuleGen(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		prepareForTestWithFsgenBuildComponents,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductCopyFiles = []string{
				"frameworks/base/config/preloaded-classes:system/etc/preloaded-classes",
				"frameworks/base/data/keyboards/Vendor_0079_Product_0011.kl:system/usr/keylayout/subdir/Vendor_0079_Product_0011.kl",
				"frameworks/base/data/keyboards/Vendor_0079_Product_18d4.kl:system/usr/keylayout/subdir/Vendor_0079_Product_18d4.kl",
				"some/non/existing/file.txt:system/etc/file.txt",
				"device/sample/etc/apns-full-conf.xml:product/etc/apns-conf.xml:google",
				"device/sample/etc/apns-full-conf.xml:product/etc/apns-conf-2.xml",
				"device/sample/etc/apns-full-conf.xml:system/foo/file.txt",
				"device/sample/etc/apns-full-conf.xml:system/foo/apns-full-conf.xml",
				"device/sample/firmware/firmware.bin:recovery/root/firmware.bin",
				"device/sample/firmware/firmware.bin:recovery/root/firmware-2.bin",
				"device/sample/firmware/firmware.bin:recovery/root/lib/firmware/firmware.bin",
				"device/sample/firmware/firmware.bin:recovery/root/lib/firmware/firmware-2.bin",
				"packages/services/Car/car_product/init/init.car.rc:root/init.car.rc",
			}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType: "ext4",
						BuildingImage:       true,
					},
				}
		}),
		prepareMockRamdiksNodeList,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
			"frameworks/base/config/preloaded-classes":                   nil,
			"frameworks/base/data/keyboards/Vendor_0079_Product_0011.kl": nil,
			"frameworks/base/data/keyboards/Vendor_0079_Product_18d4.kl": nil,
			"device/sample/etc/apns-full-conf.xml":                       nil,
			"device/sample/firmware/firmware.bin":                        nil,
			"packages/services/Car/car_product/init/init.car.rc":         nil,
		}),
	).RunTest(t)

	// check generated prebuilt_* module type install path and install partition
	generatedModule := result.ModuleForTests(t, "system-frameworks_base_config-system_etc-0", "android_arm64_armv8-a").Module()
	etcModule := generatedModule.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have etc install path",
		"etc",
		etcModule.BaseDir(),
	)
	android.AssertBoolEquals(
		t,
		"module expected to be installed in system partition",
		true,
		!generatedModule.InstallInProduct() &&
			!generatedModule.InstallInVendor() &&
			!generatedModule.InstallInSystemExt(),
	)

	// check generated prebuilt_* module specifies correct relative_install_path property
	generatedModule = result.ModuleForTests(t, "system-frameworks_base_data_keyboards-system_usr_keylayout_subdir-0", "android_arm64_armv8-a").Module()
	etcModule = generatedModule.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to set correct relative_install_path properties",
		"subdir",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module sets correct srcs
	eval := generatedModule.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"Vendor_0079_Product_0011.kl",
		getModuleProp(generatedModule, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 2 {
					return srcs[0]
				}
			}
			return ""
		}),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"Vendor_0079_Product_18d4.kl",
		getModuleProp(generatedModule, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 2 {
					return srcs[1]
				}
			}
			return ""
		}),
	)

	// check that prebuilt_* module is not generated for non existing source file
	android.AssertStringEquals(
		t,
		"prebuilt_* module not generated for non existing source file",
		"",
		strings.Join(result.ModuleVariantsForTests("system-some_non_existing-etc-0"), ","),
	)

	// check that duplicate src file can exist in PRODUCT_COPY_FILES and generates separate modules
	generatedModule0 := result.ModuleForTests(t, "product-device_sample_etc-product_etc-0", "android_arm64_armv8-a").Module()
	generatedModule1 := result.ModuleForTests(t, "product-device_sample_etc-etc-1", "android_arm64_armv8-a").Module()

	// check that generated prebuilt_* module sets correct srcs and dsts property
	eval = generatedModule0.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"apns-full-conf.xml",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 1 {
					return srcs[0]
				}
			}
			return ""
		}),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct dsts property",
		"apns-conf.xml",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) == 1 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	// check that generated prebuilt_* module sets correct srcs and dsts property
	eval = generatedModule1.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"apns-full-conf.xml",
		getModuleProp(generatedModule1, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 1 {
					return srcs[0]
				}
			}
			return ""
		}),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct dsts property",
		"apns-conf-2.xml",
		getModuleProp(generatedModule1, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) == 1 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	generatedModule0 = result.ModuleForTests(t, "system-device_sample_etc-system_foo-0", "android_common").Module()
	generatedModule1 = result.ModuleForTests(t, "system-device_sample_etc-foo-1", "android_common").Module()

	// check that generated prebuilt_* module sets correct srcs and dsts property
	eval = generatedModule0.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"apns-full-conf.xml",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 1 {
					return srcs[0]
				}
			}
			return ""
		}),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct dsts property",
		"foo/file.txt",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) == 1 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	// check generated prebuilt_* module specifies correct install path and relative install path
	etcModule = generatedModule1.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have . install path",
		".",
		etcModule.BaseDir(),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct relative_install_path properties",
		"foo",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module sets correct srcs
	eval = generatedModule1.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct srcs property",
		"apns-full-conf.xml",
		getModuleProp(generatedModule1, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltEtcProperties); ok {
				srcs := p.Srcs.GetOrDefault(eval, nil)
				if len(srcs) == 1 {
					return srcs[0]
				}
			}
			return ""
		}),
	)

	generatedModule0 = result.ModuleForTests(t, "recovery-device_sample_firmware-recovery_root-0", "android_recovery_arm64_armv8-a").Module()
	generatedModule1 = result.ModuleForTests(t, "recovery-device_sample_firmware-1", "android_recovery_common").Module()

	// check generated prebuilt_* module specifies correct install path and relative install path
	etcModule = generatedModule0.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have . install path",
		".",
		etcModule.BaseDir(),
	)
	android.AssertStringEquals(
		t,
		"module expected to set empty relative_install_path properties",
		"",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module don't set dsts
	eval = generatedModule0.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to not set dsts property",
		"",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) != 0 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	// check generated prebuilt_* module specifies correct install path and relative install path
	etcModule = generatedModule1.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have . install path",
		".",
		etcModule.BaseDir(),
	)
	android.AssertStringEquals(
		t,
		"module expected to set empty relative_install_path properties",
		"",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module sets correct dsts
	eval = generatedModule1.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct dsts property",
		"firmware-2.bin",
		getModuleProp(generatedModule1, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) == 1 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	generatedModule0 = result.ModuleForTests(t, "recovery-device_sample_firmware-recovery_root_lib_firmware-0", "android_recovery_common").Module()
	generatedModule1 = result.ModuleForTests(t, "recovery-device_sample_firmware-lib_firmware-1", "android_recovery_common").Module()

	// check generated prebuilt_* module specifies correct install path and relative install path
	etcModule = generatedModule0.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have . install path",
		".",
		etcModule.BaseDir(),
	)
	android.AssertStringEquals(
		t,
		"module expected to set correct relative_install_path properties",
		"lib/firmware",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module sets correct srcs
	eval = generatedModule0.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to not set dsts property",
		"",
		getModuleProp(generatedModule0, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) != 0 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	// check generated prebuilt_* module specifies correct install path and relative install path
	etcModule = generatedModule1.(*etc.PrebuiltEtc)
	android.AssertStringEquals(
		t,
		"module expected to have . install path",
		".",
		etcModule.BaseDir(),
	)
	android.AssertStringEquals(
		t,
		"module expected to set empty relative_install_path properties",
		"",
		etcModule.SubDir(),
	)

	// check that generated prebuilt_* module sets correct srcs
	eval = generatedModule1.ConfigurableEvaluator(android.PanickingConfigAndErrorContext(result.TestContext))
	android.AssertStringEquals(
		t,
		"module expected to set correct dsts property",
		"lib/firmware/firmware-2.bin",
		getModuleProp(generatedModule1, func(actual interface{}) string {
			if p, ok := actual.(*etc.PrebuiltDstsProperties); ok {
				dsts := p.Dsts.GetOrDefault(eval, nil)
				if len(dsts) == 1 {
					return dsts[0]
				}
			}
			return ""
		}),
	)

	generatedModule0 = result.ModuleForTests(t, "system-packages_services_Car_car_product_init-root-0", "android_arm64_armv8-a").Module()

	android.AssertBoolEquals(
		t,
		"module expected to set install_in_root property",
		true,
		getModuleProp(generatedModule0, func(actual interface{}) bool {
			if p, ok := actual.(*etc.PrebuiltRootProperties); ok {
				return proptools.Bool(p.Install_in_root)
			}
			return false
		}),
	)
}

func TestPartitionOfOverrideModules(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		android.PrepareForTestWithNamespace,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}`),
			"mynamespace/Android.bp": []byte(`
			soong_namespace{
			}
			android_app {
				name: "system_ext_app_in_namespace",
				system_ext_specific: true,
				platform_apis: true,
			}`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.NamespacesToExport = []string{"mynamespace"}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"system_ext_override_app", "system_ext_override_app_in_namespace"})
		}),
	).RunTestWithBp(t, `
android_app {
	name: "system_ext_app",
	system_ext_specific: true,
	platform_apis: true,
}
override_android_app {
	name: "system_ext_override_app",
	base: "system_ext_app",
}
override_android_app {
	name: "system_ext_override_app_in_namespace",
	base: "//mynamespace:system_ext_app_in_namespace",
}
`)
	resolvedDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["system_ext"]
	_, overrideAppInSystemExt := (*resolvedDeps)["system_ext_override_app"]
	android.AssertBoolEquals(t, "Override app should be added to the same partition as the `base`", true, overrideAppInSystemExt)
	_, overrideAppInSystemExt = (*resolvedDeps)["system_ext_override_app_in_namespace"]
	android.AssertBoolEquals(t, "Override app should be added to the same partition as the `base`", true, overrideAppInSystemExt)
}

func TestReportsAllNamespaceConflictsTogether(t *testing.T) {
	android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		prepareMockRamdiksNodeList,
		android.PrepareForTestWithNamespace,
		phony.PrepareForTestWithPhony,
		android.FixtureMergeMockFs(android.MockFS{
			"build/soong/fsgen/Android.bp": []byte(`
				soong_filesystem_creator {
					name: "foo",
				}`),
			"namespace_a/Android.bp": []byte(`
				soong_namespace {}
				phony { name: "conflict_one" }
				phony { name: "conflict_two" }
			`),
			"namespace_b/Android.bp": []byte(`
				soong_namespace {}
				phony { name: "conflict_one" }
				phony { name: "conflict_two" }
			`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.NamespacesToExport = []string{"namespace_a", "namespace_b"}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet =
				createProductPackagesSet([]string{"conflict_one", "conflict_two"})
		}),
	).ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
		`(?s)found modules in multiple namespaces.*"conflict_one" in system partition: namespace_a, namespace_b.*"conflict_two" in system partition: namespace_a, namespace_b`,
	)).RunTest(t)
}

func TestCrossPartitionRequiredModules(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		android.PrepareForTestWithNamespace,
		phony.PrepareForTestWithPhony,
		etc.PrepareForTestWithPrebuiltEtc,
		android.PrepareForTestWithHostTools("conv_linker_config"),
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"mynamespace/default-permissions.xml":        nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}`),
			"mynamespace/Android.bp": []byte(`
			soong_namespace{
			}
			android_app {
				name: "some_app_in_namespace",
				product_specific: true,
				required: ["some-permissions"],
				platform_apis: true,
			}
			prebuilt_etc {
				name: "some-permissions",
				sub_dir: "default-permissions",
				src: "default-permissions.xml",
				filename_from_src: true,
				system_ext_specific: true,
			}
`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.NamespacesToExport = []string{"mynamespace"}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"some_app_in_namespace"})
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system_ext": {
						BuildingImage: true,
					},
				}
		}),
	).RunTestWithBp(t, `
		phony {
			name: "com.android.vndk.v34",
		}
		phony {
			name: "com.android.vndk.v33",
		}
		phony {
			name: "com.android.vndk.v32",
		}
		phony {
			name: "com.android.vndk.v31",
		}
		phony {
			name: "com.android.vndk.v30",
		}
		phony {
			name: "file_contexts_bin_gen",
		}
		phony {
			name: "notice_xml_system_ext",
		}
	`)
	systemExtFilesystemModule := result.ModuleForTests(t, "test_product_generated_system_ext_image", "android_common")
	systemExtStagingDirImplicitDeps := systemExtFilesystemModule.Output("staging_dir.timestamp").Implicits
	android.AssertStringDoesContain(t,
		"system_ext staging dir expected to contain cross partition require deps",
		strings.Join(systemExtStagingDirImplicitDeps.Strings(), " "),
		"mynamespace/some-permissions/android_arm64_armv8-a/default-permissions.xml",
	)
}

func TestOverriddenDepsAreAddedToFilesystemModuleOverriddenDeps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		prepareForTestWithDefaultSystemDeps,
		android.PrepareForTestWithHostTools("conv_linker_config"),
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
			"A.java": nil,
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"libbar", "libbaz"})
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType: "ext4",
						BuildingImage:       true,
					},
					"vendor": {
						BoardFileSystemType: "ext4",
						BuildingImage:       true,
					},
				}
		}),
	).RunTestWithBp(t, `
java_library {
	name: "libfoo",
	srcs: ["A.java"],
	installable: true,
}
android_app {
	name: "libbar",
	srcs: ["A.java"],
	required: ["libfoo"],
	installable: true,
	platform_apis: true,
}
java_library {
	name: "libbaz",
	overrides: ["libfoo"],
	vendor: true,
	sdk_version: "current",
	srcs: ["A.java"],
	installable: true,
}
	`)
	systemImg := result.ModuleForTests(t, "test_product_generated_system_image", "android_common")
	var packagingProps android.PackagingProperties
	for _, prop := range systemImg.Module().GetProperties() {
		if packagingPropStruct, ok := prop.(*android.PackagingProperties); ok {
			packagingProps = *packagingPropStruct
		}
	}

	android.AssertStringListContains(t, "Overridden module expected to be in overridden_deps", packagingProps.Overridden_deps, "libfoo")

	systemImgStagingDirImplicitDeps := strings.Join(systemImg.Output("staging_dir.timestamp").Implicits.Strings(), " ")
	android.AssertStringDoesContain(t, "system image should install libbar", systemImgStagingDirImplicitDeps, "libbar.apk")
	android.AssertStringDoesNotContain(t, "system image should not install libfoo", systemImgStagingDirImplicitDeps, "libfoo.jar")
}

func TestCrossPartitionSharedLibDeps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		cc.PrepareForTestWithCcBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		android.PrepareForTestWithNamespace,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
		`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"system_ext_bin"})
		}),
	).RunTestWithBp(t, `
cc_binary {
	name: "system_ext_bin",
	shared_libs: ["system_lib"],
	system_ext_specific: true,
}
cc_library_shared {
	name: "system_lib",
}
`)
	resolvedDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["system"]
	xPartitionSharedLib := (*resolvedDeps)["system_lib"]
	android.AssertIntEquals(
		t,
		"Expected single arch variant of cross partition shared lib dependency",
		1,
		len(xPartitionSharedLib.Arch),
	)
	android.AssertStringEquals(
		t,
		"Expected primary arch variant of cross partition shared lib dependency",
		"arm64",
		xPartitionSharedLib.Arch[0].String(),
	)
}

func TestRemoveOverriddenTransitiveDeps(t *testing.T) {
	t.Run("case 1", func(t *testing.T) {
		result := android.GroupFixturePreparers(
			android.PrepareForIntegrationTestWithAndroid,
			android.PrepareForTestWithAndroidBuildComponents,
			android.PrepareForTestWithAllowMissingDependencies,
			prepareForTestWithFsgenBuildComponents,
			java.PrepareForTestWithJavaBuildComponents,
			prepareMockRamdiksNodeList,
			android.FixtureMergeMockFs(android.MockFS{
				"A.java": nil,
				"build/soong/fsgen/Android.bp": []byte(`
				soong_filesystem_creator {
					name: "filesystem_creator",
				}
				`),
			}),
			android.FixtureModifyConfig(func(config android.Config) {
				config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"bar", "baz"})
			}),
		).RunTestWithBp(t, `
	java_library {
		name: "foo",
		product_specific: true,
		srcs: ["A.java"],
	}
	java_library {
		name: "bar",
		vendor: true,
		required: ["foo"],
		srcs: ["A.java"],
		sdk_version: "current",
	}
	java_library {
		name: "baz",
		overrides: ["bar"],
		srcs: ["A.java"],
	}
	java_library {
		name: "qux",
		required: ["foo"],
		srcs: ["A.java"],
	}
		`)
		resolvedProductDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["product"]
		_, fooInDeps := (*resolvedProductDeps)["foo"]
		android.AssertBoolEquals(t, "foo should not be in deps", false, fooInDeps)
	})

	t.Run("case 2", func(t *testing.T) {
		result := android.GroupFixturePreparers(
			android.PrepareForIntegrationTestWithAndroid,
			android.PrepareForTestWithAndroidBuildComponents,
			android.PrepareForTestWithAllowMissingDependencies,
			prepareForTestWithFsgenBuildComponents,
			java.PrepareForTestWithJavaBuildComponents,
			prepareMockRamdiksNodeList,
			android.FixtureMergeMockFs(android.MockFS{
				"A.java": nil,
				"build/soong/fsgen/Android.bp": []byte(`
				soong_filesystem_creator {
					name: "filesystem_creator",
				}
				`),
			}),
			android.FixtureModifyConfig(func(config android.Config) {
				config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"bar", "baz", "qux"})
			}),
		).RunTestWithBp(t, `
	java_library {
		name: "foo",
		product_specific: true,
		srcs: ["A.java"],
	}
	java_library {
		name: "bar",
		vendor: true,
		required: ["foo"],
		srcs: ["A.java"],
		sdk_version: "current",
	}
	java_library {
		name: "baz",
		overrides: ["bar"],
		srcs: ["A.java"],
	}
	java_library {
		name: "qux",
		required: ["foo"],
		srcs: ["A.java"],
	}
		`)
		resolvedProductDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["product"]
		_, fooInDeps := (*resolvedProductDeps)["foo"]
		android.AssertBoolEquals(t, "foo should be in deps", true, fooInDeps)
	})
}

func TestVbmetaGenerationWithCustomPartitions(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		cc.PrepareForTestWithCcBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
		`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.CustomImagesPartitions = []string{"custom1"}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BuildingVbmetaImage = true
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType:           "ext4",
						BuildingImage:                 true,
						BoardAvbKeyPath:               "external/avb/test/data/testkey_rsa4096.pem",
						BoardAvbRollbackIndexLocation: "1",
					},
					"custom1": {
						BoardAvbKeyPath: "external/avb/test/data/testkey_rsa4096.pem",
					},
				}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BoardAvbEnable = true
		}),
	).RunTestWithBp(t, `
		android_filesystem {
			name: "custom1",
			use_avb: true,
			avb_private_key: "external/avb/test/data/testkey_rsa4096.pem",
			rollback_index_location: 5,
		}
	`)

	generatedVbmetaImage := result.ModuleForTests(t, "test_product_generated_vbmeta_image", "android_common").Output("vbmeta.img")
	vbmetaImageCommand := generatedVbmetaImage.RuleParams.Command

	android.AssertStringDoesContain(t, "system chained partition must exist with property appending", vbmetaImageCommand, "--chain_partition system:1:out/soong/.intermediates/build/soong/fsgen/test_product_generated_system_image/android_common/system.avbpubke")
	android.AssertStringDoesContain(t, "avb enabled custom partition must be included as chained partition", vbmetaImageCommand, "--chain_partition custom1:5:out/soong/.intermediates/custom1/android_common/custom1.avbpubkey")
}

func TestVbmetaGenerationWithPvmfw(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		cc.PrepareForTestWithCcBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
		`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BuildingVbmetaImage = true
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BoardUsesPvmfwImage = true
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardFileSystemType:           "ext4",
						BuildingImage:                 true,
						BoardAvbKeyPath:               "external/avb/test/data/testkey_rsa4096.pem",
						BoardAvbRollbackIndexLocation: "1",
					},
				}
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BoardAvbEnable = true
		}),
	).RunTestWithBp(t, `
		raw_binary {
			name: "pvmfw_bin",
		}
		bootimg {
			name: "pvmfw_img",
			kernel_prebuilt: ":pvmfw_bin",
			header_version: "3",
			use_avb: true,
		}
	`)

	generatedVbmetaImage := result.ModuleForTests(t, "test_product_generated_vbmeta_image", "android_common").Output("vbmeta.img")
	vbmetaImageCommand := generatedVbmetaImage.RuleParams.Command

	android.AssertStringDoesContain(t,
		"avb enabled pvmfw partition must be included",
		vbmetaImageCommand,
		"--include_descriptors_from_image out/soong/.intermediates/pvmfw_img/android_arm64_armv8-a/pvmfw_img.img",
	)
}

func TestStageDeviceFiles(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		filesystem.PrepareForTestWithAndroidDeviceComponents,
		prepareForTestWithFsgenBuildComponents,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductCopyFiles = []string{
				"source/dir/my_staged_file:my_staged_file",
				"source/another_file:another_file",
				"source/file3:system/etc/file3", // This should not be staged
			}
		}),
		android.FixtureMergeMockFs(android.MockFS{
			"source/dir/my_staged_file": nil,
			"source/another_file":       nil,
			"source/file3":              nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
			`),
		}),
	).RunTest(t)

	deviceModule := result.ModuleForTests(t, "test_product_generated_device", "android_arm64_armv8-a").Module()

	var deviceProps *filesystem.DeviceProperties
	for _, prop := range deviceModule.GetProperties() {
		if p, ok := prop.(*filesystem.DeviceProperties); ok {
			deviceProps = p
			break
		}
	}

	if deviceProps == nil {
		t.Fatal("Could not find DeviceProperties on generated android_device module")
	}

	expected := []filesystem.StageDeviceFilePairProp{
		{Src: proptools.StringPtr("source/another_file"), Dst: proptools.StringPtr("another_file")},
		{Src: proptools.StringPtr("source/dir/my_staged_file"), Dst: proptools.StringPtr("my_staged_file")},
	}

	android.AssertDeepEquals(t, "Stage_device_files", expected, deviceProps.Stage_device_files)

	// Verify that the files are copied to the staging dir
	allOutputs := result.ModuleForTests(t, "test_product_generated_device", "android_arm64_armv8-a").AllOutputs()
	allOutputsString := strings.Join(allOutputs, " ")

	android.AssertStringDoesContain(t, "staging dir contains expected files", allOutputsString, "my_staged_file")
	android.AssertStringDoesContain(t, "staging dir contains expected files", allOutputsString, "another_file")
	android.AssertStringDoesNotContain(t, "staging dir does not contain arbitrary subdir file", allOutputsString, "file3")
}

func TestCrossPartitionRequiredDepsOfPhony(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		android.PrepareForTestWithAllowMissingDependencies,
		prepareForTestWithFsgenBuildComponents,
		cc.PrepareForTestWithCcBuildComponents,
		java.PrepareForTestWithJavaBuildComponents,
		prepareMockRamdiksNodeList,
		android.PrepareForTestWithNamespace,
		phony.PrepareForTestWithPhony,
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
			"build/soong/fsgen/Android.bp": []byte(`
			soong_filesystem_creator {
				name: "foo",
			}
		`),
		}),
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.ProductPackagesSet = createProductPackagesSet([]string{"myphony"})
		}),
	).RunTestWithBp(t, `
// myphony has a required dependency on a system_ext binary,
// but does not set system_ext_specific to true.
phony {
	name: "myphony",
	required: ["system_ext_bin"],
}
cc_binary {
	name: "system_ext_bin",
	shared_libs: ["system_lib"],
	system_ext_specific: true,
}
`)
	resolvedDeps := result.TestContext.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps["system_ext"]
	_, exists := (*resolvedDeps)["system_ext_bin"]
	android.AssertBoolEquals(
		t,
		"Expected fsgen to add cross partition required dep of myphony",
		true,
		exists,
	)
}
