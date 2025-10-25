# read csv file
import pandas as pd

def read_csv_file(file_path):
    data = pd.read_csv(file_path)
    return data

def load_all_instances(file_path):
    data = pd.read_csv(file_path, sep=';')
    return data

file_path = 'result_B.csv'

data = read_csv_file(file_path)
print(data.head())


# grouped summary table for objective per method
summary = (
    data.groupby('method')['objective']
    .agg(min_obj='min', max_obj='max', mean_obj='mean')
    .reset_index()
)

# round numeric columns for nicer display
for col in ['min_obj', 'max_obj', 'mean_obj']:
    summary[col] = summary[col].round(3)

print(summary.to_string(index=False))

# optionally save the prepared table to CSV
summary.to_csv('summary_objectives_by_method.csv', index=False)

# load all instances from csv
instance_path ='TSPA.csv'
instances = load_all_instances(instance_path)
print(instances.head())

# plot all instances on the plot the cost column should be show by size of the point
import matplotlib.pyplot as plt
import numpy as np
def plot_instances(data):
    coords = data.iloc[:, [0, 1]].values
    costs = data.iloc[:, 2].values
    sizes = costs / max(costs) * 100

    plt.figure(figsize=(10, 6))
    plt.scatter(coords[:, 0], coords[:, 1], color="blue", s=sizes)

    for i, point in enumerate(coords):
        plt.text(point[0], point[1], str(i), fontsize=9, ha="right")

    plt.title("TSP Instances")
    plt.xlabel("X Coordinate")
    plt.ylabel("Y Coordinate")
    plt.grid(True)
    plt.show()

# plot_instances(instances)


# visualize best path for each method with minimum objective value
def visualize_best_paths(data, instances):
    methods = data['method'].unique()
    coords = instances.iloc[:, [0, 1]].values

    for method in methods:
        method_data = data[data['method'] == method]
        best_row = method_data.loc[method_data['objective'].idxmin()]
        path = list(map(int, best_row['final_selected'].split(';')))
        title = f"Best path ({method}): objective={best_row['objective']:.1f}"

        plt.figure(figsize=(10, 6))
        plt.scatter(coords[:, 0], coords[:, 1], color="blue", s=20)

        for i, point in enumerate(coords):
            plt.text(point[0], point[1], str(i), fontsize=9, ha="right")

        path_coords = coords[path + [path[0]]]
        plt.plot(
            path_coords[:, 0], path_coords[:, 1], color="red", linestyle="-", marker="o"
        )

        plt.title(title)
        plt.xlabel("X Coordinate")
        plt.ylabel("Y Coordinate")
        plt.grid(True)
        # plt.show()
        # save plot as png
        plt.savefig(f"best_path_{method.replace(':', '_')}.png")

visualize_best_paths(data, instances)