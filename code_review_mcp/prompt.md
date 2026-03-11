Persona:
you are a mcp (model context protocol) maker, who already has a lot of experience in making mcp, and you are very good at it. you have made many mcp before, and you know how to make a good mcp. you are also very creative, and you can come up with new ideas for mcp. you are also very patient, and you can take your time to make a good mcp.

Context:
Currently i was doing code review many projects, and i found manually ask copilot to review code is very time consuming, and i want to make a mcp to help me review code faster. i want to make a mcp that can automatically review code for me, and give me feedback on the code. i want the mcp to be able to understand the code, and give me feedback on the code quality, readability, and maintainability. i also want the mcp to be able to give me suggestions on how to improve the code.

What usually happen is, every pull request being raised and someone ask to me to review the code, i will ask the copilot to review from feature branch to target branch, and then i help copilot add more context like code-review-copilot-instruction.md for the instruction and copilot-instructions.md for the general project guidelines, and then copilot will give me the review comments. but this process is very time consuming, and i want to automate this process.

Task:
I need you to create the mcp for me to automate the code review process. The arguments of the mcp should be:
- feature_branch: the branch that contains the new code changes that need to be reviewed.
- target_branch: the branch that the feature branch is being merged into, usually the main or master branch.
- code_review_instruction: the instruction for the code review, which can be a markdown file that contains the specific guidelines and criteria for the code review.
- project_guidelines: the general guidelines for the project, which can be a markdown file that contains the coding standards, best practices,and other relevant information for the project.

the mcp should be able to give the caller list of review comments based on the code changes in the feature branch compared to the target branch, and the instructions provided in the code_review_instruction and project_guidelines. The mcp should also be able to provide suggestions for improving the code based on the review comments.

but mcp need to list down first to the caller every comment that might be raised by copilot, and i as a user will review the list which comment that wort to be raised, and which comment that may can be skipped, and then mcp will only raise the comment that i choose to raise. this way i can have more control over the review process, and i can also save time by skipping comments that are not relevant or important.

after that mcp need to post the comment as a review comment directly on the pull request, so we might need to integrate with the version control system (like GitHub, GitLab, etc.) to post the comments directly on the pull request. the mcp should also be able to handle any authentication or authorization required to post comments on the pull request.

but to make it easier for me to use, the argument that needed on this is only link pr, and then mcp will automatically extract the feature branch and target branch from the pr link, and then ask me to upload the code_review_instruction and project_guidelines, and then mcp will do the rest of the work. this way i can just provide the pr link and the instructions, and mcp will take care of the rest.

this mcp also need the config list like which project that have which code review instruction and project guidelines, so that when i provide the pr link, mcp can automatically find the corresponding code_review_instruction and project_guidelines for that project, and then use those instructions to review the code. this way i don't have to upload the instructions every time, and mcp can automatically find the instructions for me based on the pr link.

beside that config, i also need bitbucket or gitlab authentication config, so that mcp can authenticate with bitbucket and post comments on the pull request. the authentication config should include the necessary credentials and tokens to authenticate with bitbucket, and mcp should be able to handle the authentication process securely.

this mcp should be able run at my local machine first for me to try by attaching the mcp into vs code, and then later on we can think about how to deploy this mcp to a server or cloud platform for better accessibility and scalability. the mcp should be designed in a way that it can be easily integrated with vs code, and it should provide a user-friendly interface for me to interact with the mcp and review the code.

Tell me your approach first for this before you start writing the mcp. I want to make sure that we are on the same page and that the approach is feasible and efficient for automating the code review process. Please provide a detailed explanation of your approach, including the architecture, technologies, and tools you plan to use for developing this mcp.