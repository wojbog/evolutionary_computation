import pandas as pd
import numpy as np
import random
from math import ceil
import numpy.typing as npt
import matplotlib.pyplot as plt

from queue import PriorityQueue


def generate_distance_matrix():
    filename = "./TSPA.csv"

    df = pd.read_csv(filename, sep=";", header=0).reset_index(drop=True)

    coords = df.iloc[:, [0, 1]].values
    costs = df.iloc[:, 2].values

    dist_matrix = np.sqrt(
        ((coords[:, np.newaxis, :] - coords[np.newaxis, :, :]) ** 2).sum(axis=2)
    )
    dist_matrix = np.round(dist_matrix).astype(int)

    # dist_matrix[from, to]
    dist_matrix = (dist_matrix.T + costs).T

    return dist_matrix


def calculate_objective_fun(path: list[int], distance_matrix: npt.NDArray) -> int:
    cost = 0
    for i in range(0, len(path) - 1):
        cost += distance_matrix[path[i], path[i + 1]]

    cost += distance_matrix[path[-1], path[0]]

    return cost


def show_path(path: list[int]):
    filename = "./TSPA.csv"

    df = pd.read_csv(filename, sep=";", header=0).reset_index(drop=True)

    coords = df.iloc[:, [0, 1]].values
    costs = df.iloc[:, 2].values
    sizes = costs / max(costs) * 100  # Normalize costs to determine node sizes

    plt.figure(figsize=(10, 6))
    plt.scatter(coords[:, 0], coords[:, 1], color="blue", s=sizes)

    for i, point in enumerate(coords):
        plt.text(point[0], point[1], str(i), fontsize=9, ha="right")

    path_coords = coords[path + [path[0]]]
    plt.plot(
        path_coords[:, 0], path_coords[:, 1], color="red", linestyle="-", marker="o"
    )

    plt.title("TSP Path")
    plt.xlabel("X Coordinate")
    plt.ylabel("Y Coordinate")
    plt.grid(True)
    plt.show()


def tsp_greedy(dist_matrix: npt.NDArray) -> tuple[int, list[int]]:
    node_count = dist_matrix.shape[0]
    path_lenght = ceil(dist_matrix.shape[0] / 2)

    heaps = [PriorityQueue() for _ in range(node_count)]

    for node in range(node_count):
        for to in range(node_count):
            if node != to:
                heaps[node].put((dist_matrix[node, to], to))

    visited = set()
    path = []

    current_node = 0
    for i in range(path_lenght):
        print("Step:", i, "Current node:", current_node)
        visited.add(current_node)
        path.append(current_node)
        show_path(path)

        next_node = heaps[current_node].get()[1]
        while next_node in visited:
            next_node = heaps[current_node].get()[1]

        current_node = next_node

    cost = calculate_objective_fun(path, dist_matrix)

    return cost, path


distance_matrix = generate_distance_matrix()

cost, path = tsp_greedy(distance_matrix)
print("Final path:", path)
print("Final cost:", cost)

output_file = "output.csv"

with open(output_file, "w") as f:
    f.write(";\n".join(map(str, path)))

show_path(path)
