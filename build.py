#!/usr/bin/env python
import os
import shutil
import subprocess
import sys

project_dir = os.path.dirname(os.path.realpath(sys.argv[0]))
subprocess.call(['go', 'build', '-o', 'dist/maestro'], cwd=project_dir)

frontend_dir = os.path.join(project_dir, "frontend")
subprocess.call(['yarn'], cwd=frontend_dir)
subprocess.call(['yarn', 'build'], cwd=frontend_dir)

frontend_dist_from_dir = os.path.join(frontend_dir, "dist")
frontend_dist_to_dir = os.path.join(project_dir, "dist", "frontend")
shutil.copytree(frontend_dist_from_dir, frontend_dist_to_dir)
