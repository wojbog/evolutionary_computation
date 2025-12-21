```
PROCEDURE Operator1(parent1, parent2)
    n ← length(parent1)
    targetSize ← n / 2

    --------------------------------------------------
    STEP 1: Build edge sets from both parents
    --------------------------------------------------
    edges1 ← empty set
    edges2 ← empty set

    FOR i ← 0 TO n-1 DO
        a ← parent1[i]
        b ← parent1[(i+1) mod n]
        add (a, b) to edges1
    END FOR

    FOR i ← 0 TO n-1 DO
        a ← parent2[i]
        b ← parent2[(i+1) mod n]
        add (a, b) to edges2
    END FOR


    --------------------------------------------------
    STEP 2: Extract common subpaths
    --------------------------------------------------
    visited ← empty set
    subpaths ← empty list of lists

    FOR i ← 0 TO n-1 DO
        start ← parent1[i]

        IF start ∈ visited THEN
            CONTINUE
        END IF

        path ← [ start ]
        mark start as visited
        current ← start

        WHILE TRUE DO
            nextIndex ← index of current in parent1 + 1

            IF nextIndex ≥ n THEN
                BREAK
            END IF

            next ← parent1[nextIndex]
            edge ← (current, next)

            IF edge ∈ edges1 AND edge ∈ edges2 AND next ∉ visited THEN
                append next to path
                mark next as visited
                current ← next
            ELSE
                BREAK
            END IF
        END WHILE

        append path to subpaths
    END FOR


    --------------------------------------------------
    STEP 3: Add random single-node subpaths
    --------------------------------------------------
    allNodes ← set of all nodes in parent1
    remove all visited nodes from allNodes

    currentSize ← total number of nodes in subpaths

    WHILE currentSize < targetSize AND allNodes is not empty DO
        node ← randomly selected element from allNodes
        append [node] to subpaths
        remove node from allNodes
        currentSize ← currentSize + 1
    END WHILE


    --------------------------------------------------
    STEP 4: Randomly connect subpaths
    --------------------------------------------------
    randomly shuffle subpaths

    offspring ← empty list

    FOR each path in subpaths DO
        IF random number < 0.5 THEN
            reverse path
        END IF
        append path to offspring
    END FOR

    RETURN offspring
END PROCEDURE

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