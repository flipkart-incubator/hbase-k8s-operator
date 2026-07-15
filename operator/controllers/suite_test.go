/*
Copyright 2021.

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

package controllers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
	"github.com/stretchr/testify/assert"
)

// TestExample is a placeholder test to ensure the test suite compiles and runs.
func TestExample(t *testing.T) {
	assert.True(t, true, "This is a placeholder test.")
}

// resetHashStore clears the package-level hashStore map between tests to prevent cross-test state leakage.
func resetHashStore() {
	for k := range hashStore {
		delete(hashStore, k)
	}
}

// loadFixture reads a JSON fixture file and unmarshals it into target. Panics on any error to ensure tests fail fast
// rather than silently proceeding with zero-value objects.
func loadFixture(path string, target interface{}) {
	out, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read fixture %s: %v", path, err))
	}
	if err := json.Unmarshal(out, target); err != nil {
		panic(fmt.Sprintf("failed to unmarshal fixture %s: %v", path, err))
	}
}

// getMockHbaseClusterSafe loads the HbaseCluster test fixture from testdata/test_hbase_cluster.json.
func getMockHbaseClusterSafe() *kvstorev1.HbaseCluster {
	cluster := &kvstorev1.HbaseCluster{}
	loadFixture("../testdata/test_hbase_cluster.json", cluster)
	return cluster
}

// getMockHbaseTenantSafe loads the HbaseTenant test fixture from testdata/test_hbase_tenant.json.
func getMockHbaseTenantSafe() *kvstorev1.HbaseTenant {
	tenant := &kvstorev1.HbaseTenant{}
	loadFixture("../testdata/test_hbase_tenant.json", tenant)
	return tenant
}

// getInvalidConfigHbasetenantSafe loads the invalid-config HbaseTenant fixture (contains malformed XML) for negative testing.
func getInvalidConfigHbasetenantSafe() *kvstorev1.HbaseTenant {
	tenant := &kvstorev1.HbaseTenant{}
	loadFixture("../testdata/test_invalid_hbase_tenant.json", tenant)
	return tenant
}

// getMockHbaseStandalone loads the HbaseStandalone test fixture from testdata/test_hbase_standalone.json.
func getMockHbaseStandalone() *kvstorev1.HbaseStandalone {
	standalone := &kvstorev1.HbaseStandalone{}
	loadFixture("../testdata/test_hbase_standalone.json", standalone)
	return standalone
}