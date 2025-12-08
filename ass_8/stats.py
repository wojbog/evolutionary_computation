import pandas as pd
import sys

colnames = ["run","objective","seed",'path']

def get_best_from_ass7_solution(letter: str):
    csv_path = f"../ass_7/res{letter}.csv"
    data = pd.read_csv(csv_path)
    best = data.loc[data['objective'].idxmin()]
    
    objective = best["objective"]
    run = -1
    seed = -1
    path = " ".join(best["final_selected"].split(";"))
    print(path)

    row = pd.Series(
            data = {
                "run": run,
                "objective": objective,
                "seed": seed,
                "path": path,
                }
            )




    print(row)
    




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

    return len(common_nodes)



def get_best_solution(data: pd.DataFrame) -> pd.Series:
    return data.loc[data['objective'].idxmin()]



csv_name = "resA.csv"
data = pd.read_csv(csv_name, names=colnames)


def similarity_to_best(similarity_func, letter: str):
    data = pd.read_csv(csv_name, header=0)
    best_solution = get_best_solution(data)

    similarities = []
    for index, row in data.iterrows():
        sim = similarity_func(best_solution['path'], row['path'])
        if sim == 100:
            continue

        similarities.append(sim)

    print(similarities)
    assert len(similarities) == 999, f"Expected 999 similarities, got {len(similarities)}"

similarity_to_best(common_edges_similarity, "A")

