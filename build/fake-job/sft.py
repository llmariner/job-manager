import os

# Move files in the current directory to the "output" directory.
files = os.listdir("./ft-output")
for fn in files:
    os.rename("./ft-output/" + fn, f"output/{fn}")
