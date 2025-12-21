```
Operator1(parent1, parent2):
    extract edges common to both parents
    build maximal common subpaths
    add random single nodes until size = K
    randomly reverse subpaths
    concatenate subpaths
    return offspring
```
```
GreedyRegretStart(instance, start_node):
    Select start_node
    Select second node with minimum (distance + cost)

    while selected nodes < K:
        for each unselected node v:
            compute best insertion cost
            compute second-best insertion cost
            regret = second_best − best
            score = regret − best_total_cost

        choose node with maximum score
        insert node at its best position

    return tour, inSelected[]
```
```
CleanPopulation(instance, population, targetSize):
    RemoveDuplicates
    if population too large:
        RemoveWorst
    if population too small:
        generate new LS-improved random solutions
    return population
```
```
GenerateInitialPopulation(instance, size):
    while population size < size:
        with probability 0.8:
            greedy regret construction
        else:
            random + local search
    remove duplicates
    return population
```

```
Main Loop

RunMethod(instance):
    population = GenerateInitialPopulation

    start timer
    while time limit not exceeded:
        select random pairs of parents
        for each pair:
            offspring = crossover operator
            apply local search to offspring
            add offspring to population

        CleanPopulation

    return best solution found
```