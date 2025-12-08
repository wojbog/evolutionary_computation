import pandas as pd
import numpy as np
import sys
import matplotlib.pyplot as plt
import seaborn as sns

colnames = ["run", "objective", "seed", "path"]
LETTER = "B"


def get_best_from_ass7_solution(letter: str):
    csv_path = f"../ass_7/res{letter}.csv"
    data = pd.read_csv(csv_path)
    best = data.loc[data["objective"].idxmin()]

    objective = best["objective"]
    run = -1
    seed = -1
    path = " ".join(best["final_selected"].split(";"))

    row = pd.Series(
        data={
            "run": run,
            "objective": objective,
            "seed": seed,
            "path": path,
        }
    )
    return row


def common_edges_similarity(path1_str: str, path2_str: str) -> float:
    path1 = path1_str.split(" ")
    path2 = path2_str.split(" ")

    set1 = set()
    for i in range(len(path1) - 1):
        edge = (path1[i], path1[i + 1])
        set1.add(edge)
        edge_rev = (path1[i + 1], path1[i])
        set1.add(edge_rev)

    set1.add((path1[-1], path1[0]))
    set1.add((path1[0], path1[-1]))

    set2 = set()
    for i in range(len(path2) - 1):
        edge = (path2[i], path2[i + 1])
        set2.add(edge)
        edge_rev = (path2[i + 1], path2[i])
        set2.add(edge_rev)

    set2.add((path2[-1], path2[0]))
    set2.add((path2[0], path2[-1]))

    common_edges = set1.intersection(set2)


    return len(common_edges) // 2


def common_nodes_similarity(path1_str: str, path2_str: str) -> float:
    path1 = path1_str.split(" ")
    path2 = path2_str.split(" ")

    set1 = set(path1)
    set2 = set(path2)

    common_nodes = set1.intersection(set2)
    all_nodes = set1.union(set2)

    return len(common_nodes) 


def get_best_solution(data: pd.DataFrame) -> pd.Series:
    idx = data["objective"].idxmin()
    row = data.loc[idx]
    return idx, row


csv_name = f"res{LETTER}.csv"
data = pd.read_csv(csv_name, names=colnames, header=0)


def similarity_to_chosen(similarity_func, compare_to, skip_index):
    data = pd.read_csv(csv_name, header=0)
    best_solution = compare_to

    objectives = []
    similarities = []
    for index, row in data.iterrows():
        similarity = similarity_func(best_solution["path"], row["path"])
        if index is not None and index == skip_index:
            continue

        similarities.append(similarity)
        objectives.append(row["objective"])


    return similarities, objectives

# def similarity_average(similarity_func, letter: str):
#     data = pd.read_csv(csv_name, header=0)

#     similarities = []
#     objectives = []

#     for index, row in data.iterrows():
#         similarities_row = []
#         for index2, row2 in data.iterrows():
#             # print(f"  comparing to {index2}")
#             if index == index2:
#                 continue

#             similarity = similarity_func(row["path"], row2["path"])
#             assert 0 <= similarity < 1.0, f"Similarity out of bounds: {similarity}"
#             similarities_row.append(similarity)

#         average_similarity = np.mean(similarities_row)
#         similarities.append(average_similarity)
#         objectives.append(row["objective"])

#     return similarities, objectives



def similarity_average(similarity_func):
    data = pd.read_csv(csv_name)

    paths = data["path"].to_numpy()
    objectives = data["objective"].to_numpy()
    n = len(paths)

    # Preallocate similarity matrix
    sim_matrix = np.empty((n, n), dtype=float)

    for i in range(n):
        for j in range(n):
            if i == j:
                sim_matrix[i, j] = np.nan
            else:
                s = similarity_func(paths[i], paths[j])
                sim_matrix[i, j] = s

    similarities = np.nanmean(sim_matrix, axis=1)

    return similarities.tolist(), objectives.tolist()
    


def plot_similarity_vs_objective(similarities, objectives, title: str, filename: str):
    print("FILENAME:" , filename)
    plt.figure(figsize=(10, 6))
    sns.scatterplot(x=objectives, y=similarities)
    plt.ylabel("Similarity")
    plt.xlabel("Objective Value")
    
    # plot the line of best fit
    z = np.polyfit(objectives, similarities, 1)
    p = np.poly1d(z)
    plt.plot(objectives, p(objectives), "r--")

    plt.title(title)
    plt.grid(True)
    plt.savefig(filename)
    plt.close()
        

for func in [common_nodes_similarity, common_edges_similarity]: 
    print("Calculating similarities to best solution for function:", func.__name__)
    skip_index, best_solution = get_best_solution(data)
    sim_chosen, obj_chosen = similarity_to_chosen(func, best_solution, skip_index)
    corr_coef = np.corrcoef(sim_chosen, obj_chosen)[0, 1]
    plot_similarity_vs_objective(
        sim_chosen,
        obj_chosen,
        title=f"Instance {LETTER}\nSimilarity to Best Solution vs Objective ({func.__name__})\nCorrelation Coefficient: {corr_coef:.2f}",
        filename=f"{LETTER}_similarity_to_best_{func.__name__}.png",
        )

    print("Calculating similarities to ass7 solution for function:", func.__name__)
    sim_chosen, obj_chosen = similarity_to_chosen(func, get_best_from_ass7_solution(LETTER), None)
    corr_coef = np.corrcoef(sim_chosen, obj_chosen)[0, 1]
    plot_similarity_vs_objective(
        sim_chosen,
        obj_chosen,
        title=f"Instance {LETTER}\nSimilarity to LNS Solution vs Objective ({func.__name__})\nCorrelation Coefficient: {corr_coef:.2f}",
        filename=f"{LETTER}_similarity_to_LNS_{func.__name__}.png",
        )

    print("Calculating average similarities for function:", func.__name__)
    sim_avg, obj_avg = similarity_average(func)
    corr_coef = np.corrcoef(sim_avg, obj_avg)[0, 1]
    plot_similarity_vs_objective(
        sim_avg,
        obj_avg,
        title=f"Instance {LETTER}\nAverage Similarity vs Objective ({func.__name__})\nCorrelation Coefficient: {corr_coef:.2f}",
        filename=f"{LETTER}_similarity_average_{func.__name__}.png",
        )











