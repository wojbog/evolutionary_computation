import pandas as pd

colnames = ["run","objective","seed",'path']

csv_name = "resA.csv"
data = pd.read_csv(csv_name, names=colnames)
print(data)
