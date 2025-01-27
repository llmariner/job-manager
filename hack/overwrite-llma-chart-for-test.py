# This script overwrite the Chart.yaml for testing purposes.
# It replaces the repository field with the relative path to
# the deployments directory. This is for testing local code
# and char without publishing.
#
# Usage:
#   python overwrite-llma-chart-for-test.py <chart_yaml_path> <deployments_dir>

import yaml
import sys
import copy


def replace_repository_paths(conf, relative_paths):
    new_conf = copy.deepcopy(conf)
    for dep in new_conf["dependencies"]:
        if dep["name"] in relative_paths:
            dep["repository"] = relative_paths[dep["name"]]

    return new_conf


if __name__ == "__main__":
    # path to the Chart.yaml
    chart_yaml_path = sys.argv[1]
    # relative path to the deployments directory from the Chart.yaml
    deployments_dir = sys.argv[2]

    relative_paths = {
        "job-manager-server": f"file://{deployments_dir}/server/",
        "job-manager-dispatcher": f"file://{deployments_dir}/dispatcher/",
        "job-manager-syncer": f"file://{deployments_dir}/syncer/",
    }

    with open(chart_yaml_path, "r") as f:
        conf = yaml.safe_load(f)

    new_conf = replace_repository_paths(conf, relative_paths)

    with open(chart_yaml_path, "w") as f:
        yaml.dump(new_conf, f)
