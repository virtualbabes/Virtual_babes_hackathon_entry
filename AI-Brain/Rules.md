# AI SYSTEM PROTOCOL & BEHAVIOR RULES

## 1. MEMORY & STATE MANAGEMENT
Always consult and update the following files to maintain project context:
* **Task Tracking:** Use `AI-Brain/A.I_memory.md` to track ongoing tasks and global memory.
* **Bug Tracking:** Use `AI-Brain/orphan_fix_list.md` to log and track discovered or fixed orphaned code.
* **Development Pipeline:** Use `AI-Brain/ToDo.md` to track development milestones.
  * `DIR.md`
  * `What_is_this_repository.md`

## 2. WORKSPACE NAVIGAION
* **Active Context:** Always utilize the VS Code IDE features to search, navigate, and review open files. Do not guess; read the files to establish accurate context.

## 3. CODE QUALITY & EXECUTION
* **Output Standards:** Deliver clean, highly structured code with clear, purposeful comments.
* **Focused Modifications:** When handling large code block modifications, restrict your changes to a single file at a time to prevent context drift and errors.
* **Separation of Concerns:** Maintain strict architectural boundaries across all modules. 
* **Communication:** Supply solid, explicit development prompts and explain your structural logic before executing complex changes.

## 4. SECURITY PROTOCOL
* **Strict Data Isolation:** Enforce maximum security by maintaining an absolute, impenetrable separation between cryptocurrency transaction interactions and gamification data.