---
header-includes:
 - \usepackage{fvextra}
 - \DefineVerbatimEnvironment{Highlighting}{Verbatim}{fontsize=\scriptsize,breaklines,breakanywhere=true,commandchars=\\\{\}}
author: Carlo Götz
title: Hibercounter
subtitle: Match Line; Play Sound
---

# Where?

github.com/c-goetz/hibercounter

# What?

- matches each line on `stdin` against multiple regexes
- first matching regex triggers a sound
- simple "synth engine" for custom sounds

# Why?

- process logs during development without looking at them
    - useful to get an overview how many DB/HTTP requests are sent
- humans can process quite a lot of audio

# Demo

Demo

# Features

- simple "synth engine"
    - oscillator -> ADR envelope
    ```
             -\                   
      A     /  \  D               
          -/    \                 
         /       \                
        /         --\    R        
      -/             --\          
     /                  --\       
    /                      --     
    ```
- hot reload config (WIP)

# TODO

- fix config hot reloading
- write tests
- factor code
- make `geiger` voice sound like its name
