package state

// GetBranchingStrategies returns a list of supported branching strategies for education
func GetBranchingStrategies() []BranchingStrategy {
	return []BranchingStrategy{
		{
			ID:          "github-flow",
			Name:        "GitHub Flow",
			Description: "A lightweight, branch-based workflow. Ideal for projects that follow a continuous delivery model.",
			MainBranch:  "main",
			FlowSteps: []string{
				"1. All code in the 'main' branch should always be deployable.",
				"2. To work on something new, create a descriptive branch off of 'main'.",
				"3. Commit to that branch locally and regularly push to the server.",
				"4. Open a Pull Request to discuss your changes.",
				"5. Merge into 'main' once reviewed and tested.",
			},
		},
		{
			ID:          "git-flow",
			Name:        "Git Flow",
			Description: "A robust framework for managing large-scale projects with scheduled releases.",
			MainBranch:  "master",
			FlowSteps: []string{
				"1. 'master' stores the official release history.",
				"2. 'develop' serves as an integration branch for features.",
				"3. Feature branches are used for new features (forked from 'develop').",
				"4. Release branches prepare for a new production release.",
				"5. Hotfix branches quickly patch production releases.",
			},
		},
		{
			ID:          "trunk-based",
			Name:        "Trunk-Based Development",
			Description: "A branching model where all developers work on a single branch ('trunk'), performing small, frequent updates.",
			MainBranch:  "main",
			FlowSteps: []string{
				"1. Developers push directly to 'main' or use very short-lived feature branches.",
				"2. Avoid long-lived branches to minimize merge pain.",
				"3. High level of automated testing requirement.",
				"4. Feature flags used to decouple deployment from release.",
			},
		},
	}
}
