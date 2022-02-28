# Copyright 2018- The Pixie Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

load("@bazel_tools//tools/build_defs/repo:git.bzl", "new_git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load(":repository_locations.bzl", "GIT_REPOSITORY_LOCATIONS", "LOCAL_REPOSITORY_LOCATIONS", "REPOSITORY_LOCATIONS")

# Make all contents of an external repository accessible under a filegroup.
# Used for external HTTP archives, e.g. cares.
BUILD_ALL_CONTENT = """filegroup(name = "all", srcs = glob(["**"]), visibility = ["//visibility:public"])"""

def _http_archive_repo_impl(name, **kwargs):
    # `existing_rule_keys` contains the names of repositories that have already
    # been defined in the Bazel workspace. By skipping repos with existing keys,
    # users can override dependency versions by using standard Bazel repository
    # rules in their WORKSPACE files.
    existing_rule_keys = native.existing_rules().keys()
    if name in existing_rule_keys:
        # This repository has already been defined, probably because the user
        # wants to override the version. Do nothing.
        return

    location = REPOSITORY_LOCATIONS[name]

    # HTTP tarball at a given URL. Add a BUILD file if requested.
    http_archive(
        name = name,
        urls = location["urls"],
        sha256 = location["sha256"],
        strip_prefix = location.get("strip_prefix", ""),
        **kwargs
    )

def _git_repo_impl(name, **kwargs):
    # `existing_rule_keys` contains the names of repositories that have already
    # been defined in the Bazel workspace. By skipping repos with existing keys,
    # users can override dependency versions by using standard Bazel repository
    # rules in their WORKSPACE files.
    existing_rule_keys = native.existing_rules().keys()
    if name in existing_rule_keys:
        # This repository has already been defined, probably because the user
        # wants to override the version. Do nothing.
        return

    location = GIT_REPOSITORY_LOCATIONS[name]

    # HTTP tarball at a given URL. Add a BUILD file if requested.
    new_git_repository(
        name = name,
        remote = location["remote"],
        commit = location["commit"],
        init_submodules = True,
        recursive_init_submodules = True,
        shallow_since = location.get("shallow_since", ""),
        **kwargs
    )

def _local_repo_impl(name, **kwargs):
    # `existing_rule_keys` contains the names of repositories that have already
    # been defined in the Bazel workspace. By skipping repos with existing keys,
    # users can override dependency versions by using standard Bazel repository
    # rules in their WORKSPACE files.
    existing_rule_keys = native.existing_rules().keys()
    if name in existing_rule_keys:
        # This repository has already been defined, probably because the user
        # wants to override the version. Do nothing.
        return

    location = LOCAL_REPOSITORY_LOCATIONS[name]

    native.new_local_repository(
        name = name,
        path = location["path"],
        **kwargs
    )

def _git_repo(name, **kwargs):
    _git_repo_impl(name, **kwargs)

def _local_repo(name, **kwargs):
    _local_repo_impl(name, **kwargs)

# For bazel repos do not require customization.
def _bazel_repo(name, **kwargs):
    _http_archive_repo_impl(name, **kwargs)

# With a predefined "include all files" BUILD file for a non-Bazel repo.
def _include_all_repo(name, **kwargs):
    kwargs["build_file_content"] = BUILD_ALL_CONTENT
    _http_archive_repo_impl(name, **kwargs)

def _com_llvm_lib():
    native.new_local_repository(
        name = "com_llvm_lib",
        build_file = "bazel/external/llvm.BUILD",
        path = "/opt/clang-13.0",
    )

    native.new_local_repository(
        name = "com_llvm_lib_libcpp",
        build_file = "bazel/external/llvm.BUILD",
        path = "/opt/clang-13.0-libc++",
    )

def _cc_deps():
    # Dependencies with native bazel build files.
    _bazel_repo("com_google_protobuf", patches = ["@px//bazel/external:protobuf.patch", "@px//bazel/external:protobuf_text_format.patch"], patch_args = ["-p1"])
    _bazel_repo("com_github_grpc_grpc", patches = ["@px//bazel/external:grpc.patch"], patch_args = ["-p1"])
    _bazel_repo("com_google_benchmark")
    _bazel_repo("com_google_googletest")
    _bazel_repo("com_github_gflags_gflags")
    _bazel_repo("com_github_google_glog")
    _bazel_repo("com_google_absl")
    _bazel_repo("com_google_flatbuffers")
    _bazel_repo("org_tensorflow")
    _bazel_repo("com_github_neargye_magic_enum")
    _bazel_repo("com_github_thoughtspot_threadstacks")
    _bazel_repo("com_github_google_re2")
    _bazel_repo("com_google_boringssl")

    # Remove the pull and push directory since they depends on civet and we don't
    # want to pull in the dependency for now.
    _bazel_repo("com_github_jupp0r_prometheus_cpp")

    # Dependencies where we provide an external BUILD file.
    _bazel_repo("com_github_apache_arrow", build_file = "@px//bazel/external:arrow.BUILD")
    _bazel_repo("com_github_ariafallah_csv_parser", build_file = "@px//bazel/external:csv_parser.BUILD")
    _bazel_repo("com_github_arun11299_cpp_jwt", build_file = "@px//bazel/external:cpp_jwt.BUILD")
    _bazel_repo("com_github_cameron314_concurrentqueue", build_file = "@px//bazel/external:concurrentqueue.BUILD")
    _bazel_repo("com_github_cmcqueen_aes_min", patches = ["@px//bazel/external:aes_min.patch"], patch_args = ["-p1"], build_file = "@px//bazel/external:aes_min.BUILD")
    _bazel_repo("com_github_cyan4973_xxhash", build_file = "@px//bazel/external:xxhash.BUILD")
    _bazel_repo("com_github_nlohmann_json", build_file = "@px//bazel/external:nlohmann_json.BUILD")
    _bazel_repo("com_github_packetzero_dnsparser", build_file = "@px//bazel/external:dnsparser.BUILD")
    _bazel_repo("com_github_rlyeh_sole", build_file = "@px//bazel/external:sole.BUILD")
    _bazel_repo("com_github_serge1_elfio", build_file = "@px//bazel/external:elfio.BUILD")
    _bazel_repo("com_github_derrickburns_tdigest", build_file = "@px//bazel/external:tdigest.BUILD")
    _bazel_repo("com_github_tencent_rapidjson", build_file = "@px//bazel/external:rapidjson.BUILD")
    _bazel_repo("com_github_vinzenz_libpypa", build_file = "@px//bazel/external:libpypa.BUILD")
    _bazel_repo("com_google_double_conversion", build_file = "@px//bazel/external:double_conversion.BUILD")
    _bazel_repo("com_github_google_sentencepiece", build_file = "@px//bazel/external:sentencepiece.BUILD", patches = ["@px//bazel/external:sentencepiece.patch"], patch_args = ["-p1"])
    _bazel_repo("com_github_antlr_antlr4", build_file = "@px//bazel/external:antlr4.BUILD", patches = ["@px//bazel/external:antlr4.patch"], patch_args = ["-p1"])
    _bazel_repo("com_github_antlr_grammars_v4", build_file = "@px//bazel/external:antlr_grammars.BUILD", patches = ["@px//bazel/external:antlr_grammars.patch"], patch_args = ["-p1"])
    _bazel_repo("com_github_pgcodekeeper_pgcodekeeper", build_file = "@px//bazel/external:pgsql_grammar.BUILD", patches = ["@px//bazel/external:pgsql_grammar.patch"], patch_args = ["-p1"])
    _bazel_repo("com_github_simdutf_simdutf", build_file = "@px//bazel/external:simdutf.BUILD")
    _bazel_repo("com_github_USCiLab_cereal", build_file = "@px//bazel/external:cereal.BUILD")
    _bazel_repo("com_intel_tbb", build_file = "@px//bazel/external:tbb.BUILD")
    _bazel_repo("com_google_farmhash", build_file = "@px//bazel/external:farmhash.BUILD")
    _bazel_repo("com_github_h2o_picohttpparser", build_file = "@px//bazel/external:picohttpparser.BUILD")
    _bazel_repo("com_github_opentelemetry_proto", build_file = "@px//bazel/external:opentelemetry.BUILD")

    # Uncomment these to develop bcc and/or bpftrace locally. Should also comment out the corresponding _git_repo lines.
    # _local_repo("com_github_iovisor_bcc", build_file = "@px//bazel/external/local_dev:bcc.BUILD")
    # _local_repo("com_github_iovisor_bpftrace", build_file = "@px//bazel/external/local_dev:bpftrace.BUILD")
    _git_repo("com_github_iovisor_bcc", build_file = "@px//bazel/external:bcc.BUILD")
    _git_repo("com_github_iovisor_bpftrace", build_file = "@px//bazel/external:bpftrace.BUILD")

    # TODO(jps): For jattach, consider using a patch and directly pulling from upstream (vs. fork).
    _git_repo("com_github_apangin_jattach", build_file = "@px//bazel/external:jattach.BUILD")

    # Dependencies used in foreign cc rules (e.g. cmake-based builds)
    _include_all_repo("com_github_gperftools_gperftools")
    _include_all_repo("com_github_nats_io_natsc", patches = ["@px//bazel/external:natsc.patch"], patch_args = ["-p1"])
    _include_all_repo("com_github_libuv_libuv", patches = ["@px//bazel/external:libuv.patch"], patch_args = ["-p1"])
    _include_all_repo("com_github_libarchive_libarchive")

def list_pl_deps(name):
    repo_urls = list()
    for repo_name, repo_config in REPOSITORY_LOCATIONS.items():
        urls = repo_config["urls"]
        best_url = None
        for url in urls:
            if url.startswith("https://github.com") or best_url == None:
                best_url = url
        repo_urls.append(best_url)

    for repo_name, repo_config in GIT_REPOSITORY_LOCATIONS.items():
        remote = repo_config["remote"]
        if remote.endswith(".git"):
            remote = remote[:-len(".git")]
        repo_urls.append(remote)

    native.genrule(
        name = name,
        outs = ["{}.out".format(name)],
        cmd = 'echo "{}" > $@'.format("\n".join(repo_urls)),
        visibility = ["//visibility:public"],
    )

def pl_deps():
    _bazel_repo("bazel_skylib")
    _bazel_repo("bazel_gazelle")
    _bazel_repo("distroless")
    _bazel_repo("io_bazel_rules_go")
    _bazel_repo("io_bazel_rules_scala")
    _bazel_repo("rules_jvm_external")
    _bazel_repo("rules_foreign_cc")
    _bazel_repo("io_bazel_rules_k8s")
    _bazel_repo("io_bazel_rules_closure")
    _bazel_repo("io_bazel_rules_docker")
    _bazel_repo("rules_python")
    _bazel_repo("com_github_bazelbuild_buildtools")

    _com_llvm_lib()
    _cc_deps()
