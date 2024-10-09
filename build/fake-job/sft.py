import os

# Move files in the current directory to the "output" directory.
files = os.listdir()
for fn in files:
    if fn == "output" || fn.endswith(".py"):
        continue
    os.rename(fn, f"output/{fn}")
