import math
import random
import numpy as np
from itertools import combinations

# ---------- Utilities ----------
def euclidean_round(a, b):
    # mathematical rounding (half up)
    return int(math.floor(math.hypot(a[0]-b[0], a[1]-b[1]) + 0.5))

def compute_distance_matrix(coords):
    n = len(coords)
    D = np.zeros((n, n), dtype=int)
    for i in range(n):
        for j in range(i+1, n):
            d = euclidean_round(coords[i], coords[j])
            D[i,j] = D[j,i] = d
    return D

def cycle_length(path, D):
    return sum(D[path[i], path[(i+1)%len(path)]] for i in range(len(path)))

def total_cost(path, costs, D):
    return cycle_length(path, D) + sum(costs[i] for i in path)

def k_from_n(n): return (n+1)//2  # ceil(n/2)

# ---------- 1. Random solution ----------
def random_solution(n, k):
    nodes = random.sample(range(n), k)
    random.shuffle(nodes)
    return nodes

# ---------- 2. Nearest neighbor (append only) ----------
def nn_end(D, costs, start, k):
    n = len(D)
    path = [start]
    remaining = set(range(n)) - {start}

    while len(path) < k:
        best_j, best_delta = None, float('inf')
        for j in remaining:
            # cost of inserting at end
            if len(path) == 1:
                delta = D[path[-1], j] + D[j, path[0]] + costs[j]
            else:
                old_edge = D[path[-1], path[0]]
                new_edges = D[path[-1], j] + D[j, path[0]]
                delta = (new_edges - old_edge) + costs[j]
            if delta < best_delta:
                best_j, best_delta = j, delta
        path.append(best_j)
        remaining.remove(best_j)
    return path

# ---------- 3. Nearest neighbor (insert anywhere) ----------
def nn_insert_anywhere(D, costs, start, k):
    n = len(D)
    path = [start]
    remaining = set(range(n)) - {start}

    # second node: choose one minimizing D[start,j]+c_j
    if remaining:
        best_j = min(remaining, key=lambda j: D[start,j] + D[j,start] + costs[j])
        path.append(best_j)
        remaining.remove(best_j)

    while len(path) < k:
        best_delta, best_j, best_pos = float('inf'), None, None
        for j in remaining:
            for i in range(len(path)):
                i2 = (i+1) % len(path)
                delta = D[path[i], j] + D[j, path[i2]] - D[path[i], path[i2]] + costs[j]
                if delta < best_delta:
                    best_delta, best_j, best_pos = delta, j, i2
        path.insert(best_pos, best_j)
        remaining.remove(best_j)
    return path

# ---------- 4. Greedy cycle ----------
def greedy_cycle(D, costs, k):
    n = len(D)
    # start with edge (i,j) minimizing D[i,j] + c_i + c_j
    best_pair = min(combinations(range(n), 2),
                    key=lambda p: D[p[0], p[1]] + costs[p[0]] + costs[p[1]])
    path = [best_pair[0], best_pair[1]]
    remaining = set(range(n)) - set(path)

    while len(path) < k:
        best_delta, best_j, best_pos = float('inf'), None, None
        for j in remaining:
            for i in range(len(path)):
                i2 = (i+1) % len(path)
                delta = D[path[i], j] + D[j, path[i2]] - D[path[i], path[i2]] + costs[j]
                if delta < best_delta:
                    best_delta, best_j, best_pos = delta, j, i2
        path.insert(best_pos, best_j)
        remaining.remove(best_j)
    return path

# ---------- Wrapper to generate 200 runs ----------
def generate_greedy_solutions(coords, costs, method='random', runs=200):
    n = len(coords)
    k = k_from_n(n)
    D = compute_distance_matrix(coords)
    all_solutions = []

    if method == 'random':
        for _ in range(runs):
            path = random_solution(n, k)
            all_solutions.append((path, total_cost(path, costs, D)))

    elif method == 'nn_end':
        starts = random.sample(range(n), min(runs, n))
        for s in starts:
            path = nn_end(D, costs, s, k)
            all_solutions.append((path, total_cost(path, costs, D)))

    elif method == 'nn_anywhere':
        starts = random.sample(range(n), min(runs, n))
        for s in starts:
            path = nn_insert_anywhere(D, costs, s, k)
            all_solutions.append((path, total_cost(path, costs, D)))

    elif method == 'greedy_cycle':
        for _ in range(runs):
            path = greedy_cycle(D, costs, k)
            all_solutions.append((path, total_cost(path, costs, D)))
    else:
        raise ValueError("Unknown method")

    # Return best solution and stats
    best = min(all_solutions, key=lambda x: x[1])
    avg = np.mean([v for _, v in all_solutions])
    return best, avg, all_solutions

def import_data(filename):
    coords = []
    costs = []
    step = 0
    with open(filename, 'r') as f:
        for line in f:
            if step == 0:
                step += 1
                continue
            x, y, c = map(float, line.strip().split(';'))
            coords.append((x,y))
            costs.append(int(c))
    return coords, costs

# ---------- Example usage ----------
if __name__ == "__main__":
    # coords = [(0,0),(10,0),(10,10),(0,10),(5,5),(20,20),(15,0)]
    # costs = [5,2,3,2,10,1,8]
    coords, costs = import_data('TSPA.csv')

    for method in ['random', 'nn_end', 'nn_anywhere', 'greedy_cycle']:
        best, avg, sols = generate_greedy_solutions(coords, costs, method)
        print(f"{method:>12} | best={best[1]:5.1f} | avg={avg:5.1f}")
