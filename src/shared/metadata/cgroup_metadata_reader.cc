/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include <algorithm>
#include <fstream>
#include <memory>
#include <string>

#include "src/common/base/base.h"
#include "src/common/base/file.h"
#include "src/shared/metadata/cgroup_metadata_reader.h"
#include "src/shared/metadata/k8s_objects.h"

namespace px {
namespace md {

std::filesystem::path CGroupMetadataReader::proc_base_path_;
std::regex CGroupMetadataReader::container_id_reg;

CGroupMetadataReader::CGroupMetadataReader(const system::Config& cfg)
    : CGroupMetadataReader(cfg.sysfs_path().string()) {
  proc_base_path_ = cfg.proc_path();
  container_id_reg = std::regex("\b[0-9a-f]{64}\b");
}

CGroupMetadataReader::CGroupMetadataReader(std::string sysfs_path) {
  // Create the new path resolver.
  auto path_resolver_or_status = CGroupPathResolver::Create(sysfs_path);
  path_resolver_ = path_resolver_or_status.ConsumeValueOr(nullptr);

  if (path_resolver_or_status.ok()) {
    LOG(INFO) << absl::Substitute("Using path_resolver with configuration: $0",
                                  path_resolver_->SpecString());
    return;
  }

  // Fallback: Legacy path resolver.
  LOG(ERROR) << absl::Substitute(
      "Failed to create path resolver. Falling back to legacy path resolver. [error = $0]",
      path_resolver_or_status.ToString());

  auto legacy_path_resolver_or_status = LegacyCGroupPathResolver::Create(sysfs_path);
  legacy_path_resolver_ = legacy_path_resolver_or_status.ConsumeValueOr(nullptr);

  if (!legacy_path_resolver_or_status.ok()) {
    LOG(ERROR) << absl::Substitute(
        "Failed to create legacy path resolver. This is not recoverable. [error = $0]",
        legacy_path_resolver_or_status.ToString());
  }
}

StatusOr<std::string> CGroupMetadataReader::PodPath(PodQOSClass qos_class, std::string_view pod_id,
                                                    std::string_view container_id,
                                                    ContainerType container_type) const {
  if (path_resolver_ != nullptr) {
    return path_resolver_->PodPath(qos_class, pod_id, container_id);
  }

  if (legacy_path_resolver_ != nullptr) {
    return legacy_path_resolver_->PodPath(qos_class, pod_id, container_id, container_type);
  }

  return error::Internal("No valid cgroup path resolver.");
}

Status CGroupMetadataReader::ReadPIDs(PodQOSClass qos_class, std::string_view pod_id,
                                      std::string_view container_id, ContainerType container_type,
                                      absl::flat_hash_set<uint32_t>* pid_set) const {
  CHECK(pid_set != nullptr);

  // The container files need to be recursively read and the PIDs needs be merge across all
  // containers.

  PL_ASSIGN_OR_RETURN(std::string fpath, PodPath(qos_class, pod_id, container_id, container_type));

  std::ifstream ifs(fpath);
  if (!ifs) {
    // This might not be a real error since the pod could have disappeared.
    return error::NotFound("Failed to open file $0", fpath);
  }

  std::string line;
  while (std::getline(ifs, line)) {
    if (line.empty()) {
      continue;
    }
    int64_t pid;
    if (!absl::SimpleAtoi(line, &pid)) {
      LOG(WARNING) << absl::Substitute("Failed to parse pid file: $0", fpath);
      continue;
    }
    pid_set->emplace(pid);
  }
  return Status::OK();
}

StatusOr<std::vector<std::string>> CGroupMetadataReader::ReadContainerIds(uint32_t pid) {
  std::filesystem::path proc_path = proc_base_path_ / std::to_string(pid);
  std::ifstream ifs(proc_path);
  if (!ifs) {
    return error::NotFound("Failed to open file $0", proc_path.string());
  }

  std::string line;
  std::vector<std::string> container_ids;
  while(std::getline(ifs, line)) {
    if (line.empty()) {
      continue;
    }
    size_t idx = line.find("pids");
    if (idx == line.npos) {
      continue;
    }
    std::string container_id;
    std::cmatch m;
    auto ret = std::regex_search(line.c_str(), m, container_id_reg);
    if (ret) {
      for (auto& elem : m) {
        container_ids.push_back(elem);
      }
    } else {
      LOG(WARNING) << absl::Substitute("Failed to find container id for pid:$0, cgroup line:$1", pid, line);
    }
  }
  return container_ids;
}

}  // namespace md
}  // namespace px
