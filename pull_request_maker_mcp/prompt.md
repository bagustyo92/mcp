Persona:
you are a mcp (model context protocol) maker, who already has a lot of experience in making mcp, and you are very good at it. you have made many mcp before, and you know how to make a good mcp. you are also very creative, and you can come up with new ideas for mcp. you are also very patient, and you can take your time to make a good mcp.

Context:
for now, we manually create pull request for every code change, and we have to manually ask copilot to review the code, which is very time consuming. we want to automate this process by creating a mcp that can automatically review the code changes before raising the pull request, and give us feedback on the code quality, readability, and maintainability. we also want the mcp to be able to give us suggestions on how to improve the code, and post the review comments directly on the pull request.

Task:
the instruction for doing the code review should be follow the code_review_instruction and project_guidelines, which can be markdown files that contain the specific guidelines and criteria for the code review, as well as the general guidelines for the project. the mcp should be able to understand these instructions and use them to review the code changes in the feature branch compared to the target branch.

for this we can use existing implementation of code_review_mcp folder, which contains the code for the mcp that can review the code changes and give feedback. we can modify this implementation to fit our specific needs and requirements, and integrate it with our pull request workflow.

after able to review and give feedback on the code changes, the mcp should be able to automatically raise a pull request with the code changes, and post the pull request include the description standard. this will save us a lot of time and effort, and help us maintain a high code quality and consistency across our project.

the standard for the pull request description can be defined in a markdown file, which can include the required sections and information that should be included in the pull request description, such as the summary of the changes, the motivation for the changes, the testing done, and any other relevant information. the mcp should be able to generate a pull request description that follows this standard, and include all the necessary information. also need to configurable like before.

on the config i need to pull request description can be tuned, either comprehensive or concise, so we can choose the level of detail we want to include in the pull request description. this will allow us to have more flexibility and control over our pull request workflow, and ensure that we are providing the right amount of information for our reviewers.

also the people who will review the code changes can be configurable, so we can specify which reviewers we want to assign to the pull request, and the mcp can automatically assign the reviewers based on their expertise and availability. this will help us ensure that our code changes are being reviewed by the right people, and that we are getting the best feedback possible.

it will be more awesome if we can also directly info to the group at google chat group when the pull request is raised, so that the team can be aware of the new code changes and can review the pull request in a timely manner. this will help us improve our communication and collaboration within the team, and ensure that we are all on the same page regarding the code changes. either use personal gchat or some bot that can post the message to the group.

template message for raising ticket to gchat can be like this:

```New Pull Request: [PR Title]
Author: [Author Name]
Description: [PR Description]
Ticket: Jira Ticket Link
PR Link: Pull Request Link

please help to review and approve this pull request. @all thank you!
```

* @all should be mentioning all the members in the group, so they can get notified about the new pull request and can review it as soon as possible.
* Jira ticket link usually exist at commit message or branch name, so we can extract it from there and include it in the message to provide more context about the changes and the related ticket.

Notes:
- Last mcp creation code_review_mcp are using typescript, for now we can explore other language like go, so i can easily read and debug how we implement this tools.
- I need you to give me your idea and purpose how we able to achieve this task, and also the steps that we need to take to implement this mcp.
- I need you to write some documentation for this mcp at markdown format README.md, which can include the overview of the mcp, the features and capabilities, the installation and usage instructions, and any other relevant information. this documentation should be clear and concise, and should provide all the necessary information for users to understand and use the mcp effectively.
- about target branch should be configurable, so we can specify which branch we want to compare the code changes with, and raise the pull request to. this will allow us to have more flexibility and control over our pull request workflow, and ensure that we are always comparing our code changes with the correct branch.
- we can also consider adding some additional features to the mcp, such as the ability to automatically merge the pull request if it passes the code review and meets all the criteria, or the ability to automatically assign reviewers to the pull request based on their expertise and availability. these features can further streamline our pull request workflow and improve our code quality and consistency.
- treat this as new project under pull_request_maker_mcp folder, and we can create a new repository for this project, and we can also create a new branch for the development of this mcp, so we can keep the code changes organized and separate from the main branch. this will also allow us to easily track the progress of the development and make any necessary adjustments along the way.
- do not make any changes outside the pull_request_maker_mcp folder, as we want to keep the code changes isolated and focused on the development of this mcp. this will also help us avoid any potential conflicts or issues with other parts of the codebase, and ensure that we are only making changes that are relevant to the development of this mcp.