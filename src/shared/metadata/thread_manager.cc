//
// Created by 千陆 on 2022/4/19.
//

#include "src/shared/metadata/thread_manager.h"
#include <algorithm>
#include <fstream>

namespace px {
namespace md {

std::unique_ptr<ThreadManager> g_instance;

const uint32_t ThreadManager::kPeriodYoung;
const uint32_t ThreadManager::kPeriodOld;
const uint32_t ThreadManager::kPeriodExpired;

ThreadManager& ThreadManager::GetInstance() {
  if (g_instance == nullptr) {
    ResetInstance();
  }
  return *g_instance;
}

void ThreadManager::ResetInstance() {
  g_instance = std::make_unique<ThreadManager>();
}

std::string ThreadManager::FindCidByPid(uint32_t pid) const {
  if (pid_2_container_id_.contains(pid)) {
    return pid_2_container_id_.at(pid);
  }
  return "";
}

StatusOr<std::vector<std::string>> ThreadManager::ReadContainerIds(uint32_t pid) {
  std::filesystem::path proc_path = host_proc_path_ / std::to_string(pid);
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
    auto ret = std::regex_search(line.c_str(), m, container_id_reg_);
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

Status ThreadManager::SetCurrentPids(absl::flat_hash_set<UPID>& upids) {
  absl::flat_hash_map<uint32_t, uint32_t> new_pids;
  for (const auto& upid : upids) {
    uint32_t pid = upid.pid();
    new_pids.emplace(pid, kPeriodYoung);
    pids[pid] = kPeriodYoung;
    if (!pid_2_container_id_.contains(pid)) {
      std::vector<std::string> cids = ReadContainerIds(pid).ConsumeValueOrDie();
      if (!cids.empty()) {
        pid_2_container_id_[pid] = cids.front();
//        VLOG(1) << absl::Substitute("[ThreadManager] SetCurrentPids, pid:$0, cids:$1", pid, );
      }
    }
  }

  // clean pids map
  for (std::pair<uint32_t, uint32_t> pair : pids) {
    uint32_t pid = pair.first;
    if (new_pids.contains(pid)) {
      continue;
    }
    // process exited
    uint32_t period = --pids[pid];
    if (period <= kPeriodExpired) {
      // time to recycle
      pid_2_container_id_.erase(pid);
    } else {
      // wait to recycle
      new_pids.emplace(pid, period);
    }
  }
  pids = std::move(new_pids);
  return Status::OK();
}

}
}