# Problem Description
The problem considered in this project involves a set of nodes, each represented by three
integer values: two coordinates (x,y)(x, y)(x,y) that define the node’s position in a
two-dimensional plane, and a cost value associated with the node. The objective is to select
exactly 50% of all available nodes (if the total number of nodes is odd, the number of
selected nodes is rounded up) and construct a Hamiltonian cycle — that is, a closed path
visiting each selected node exactly once and returning to the starting node.
The goal is to minimize the sum of two components:
1. The total length of the constructed cycle, and
2. The total cost of all selected nodes
The distances between nodes are calculated using the Euclidean metric, rounded to the
nearest integer.

# Implemented Algorithms
## Greedy Heuristics with a Weighted Sum Criterion
![Greedy-heuristics-with-a-weighted-sum-criterion](Greedy-heuristics-with-a-weighted-sum-criterion.png)

## 2-Regret Insertion Heuristic
![Greedy-2-regret-heuristics](Greedy-2-regret-heuristics.png)

# Results

| Method                        | Best | Worst | Average |
|-------------------------------|:----:|:-----:|:-------:|
| 2-Regret Insertion            |      |       |         |
| Weighted Sum (α=1.00, β=1.00) |      |       |         |

## Best path for the instance A
![](./TSPA_Weighted (α=1.00,β=1.00).png)
![](./TSPA_2-Regret insertion.png)

## Best path for the instance B
![](./TSPB_Weighted (α=1.00,β=1.00).png)
![](./TSPB_2-Regret insertion.png)


# Comparison
## Instance A
| Algorithm                                | Total Distance | Total Cost   | Objective Value |
|------------------------------------------|:--------------:|:----------:|:-----------------:|
| 2-Regret Insertion                       |       21373    |  84479       |         **105852**  |
|           Weighted Sum (α=1.00, β=1.00)  |     22981      | 48127        |  **71108**          |

| NN - End                                 |                |              |                   |
| NN - Anywhere                                 |                |              |                   |
| Greedy Cycle                                 |                |              |                   |

## Instance B
| Algorithm                                | Total Distance | Total Cost   | Objective Value |
|------------------------------------------|:----------------:|:--------------:|:-----------------:|
| 2-Regret Insertion                       |     22454      | 44051        |    **66505**        |
|           Weighted Sum (α=1.00, β=1.00)  |    21152       | 25992        |  **47144**          |
| NN - End                                 |                |              |                   |
| NN - Anywhere                                 |                |              |                   |
| Greedy Cycle                                 |                |              |                   |
