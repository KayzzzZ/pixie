//
// Created by 千陆 on 2022/4/19.
//

#ifndef PIXIE_THREAD_MANAGER_H
#define PIXIE_THREAD_MANAGER_H

#include "src/common/base/base.h"
#include "src/common/system/system.h"
#include "src/shared/metadata/pids.h"

#include <memory>
#include <string>
#include <vector>
#include <regex>
#include <filesystem>

namespace px {
namespace md {

class ThreadManager : public NotCopyable {
public:
  explicit ThreadManager(system::Config& config) : container_id_reg_(std::regex("\b[0-9a-f]{64}\b")), host_proc_path_(config.proc_path()) {}
  ThreadManager() : container_id_reg_(std::regex("\b[0-9a-f]{64}\b")) {
    const system::Config& config = system::Config::GetInstance();
    host_proc_path_ = config.proc_path();
  }
  static ThreadManager& GetInstance();
  static void ResetInstance();
  ~ThreadManager() {}

  std::string FindCidByPid(uint32_t pid) const;

  Status SetCurrentPids(absl::flat_hash_set<UPID>& upids);

protected:

private:
  StatusOr<std::vector<std::string>> ReadContainerIds(uint32_t pid);
  absl::flat_hash_map<uint32_t, std::string> pid_2_container_id_;
  absl::flat_hash_map<uint32_t, uint32_t> pids;
  std::regex container_id_reg_;
  std::filesystem::path host_proc_path_;

  static const uint32_t kPeriodYoung = 3;
  static const uint32_t kPeriodOld = 2;
  static const uint32_t kPeriodExpired = 0;
};

}
}

#endif //PIXIE_THREAD_MANAGER_H
