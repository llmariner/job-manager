import os

files = os.listdir("./ft-output")
for fn in files:
    os.rename("./ft-output/" + fn, f"output/{fn}")
