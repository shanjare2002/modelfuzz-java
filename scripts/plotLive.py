import json
import matplotlib.pyplot as plt
import numpy as np
import time
import os
from pathlib import Path

def find_project_root():
    current_dir = Path.cwd()
    
    for path in [current_dir] + list(current_dir.parents):
        finalOutputs_path = path / "finalOutputs"
        if finalOutputs_path.exists() and finalOutputs_path.is_dir():
            return path
    
    return current_dir

stats_file = find_project_root() / 'output/modelfuzz/stats.json'
plt.ion()
fig = None
axes = None

def update_plots():
    global fig, axes
    
    try:
        with open(stats_file, 'r') as f:
            data = json.load(f)
    except Exception as e:
        print(f"Error loading JSON: {e}")
        return False

    coverages = data['Coverages']
    transitions = data['Transitions']
    random_traces = data['RandomTraces']
    mutated_traces = data['MutatedTraces']
    code_coverage = data.get('CodeCoverage', None)
    iterations = range(len(coverages))

    if fig is None:
        plot_rows = 3 if code_coverage else 2
        plot_cols = 2
        fig, axes = plt.subplots(plot_rows, plot_cols, figsize=(15, 12))
        axes = axes.flatten()
        
        num_plots = 5 if code_coverage else 4
        for i in range(num_plots, len(axes)):
            fig.delaxes(axes[i])
    
    for ax in axes[:5 if code_coverage else 4]:
        ax.clear()

    # 1: Coverage plot
    axes[0].plot(iterations, coverages, 'b-', linewidth=2, label='Coverage')
    axes[0].set_xlabel('Iteration')
    axes[0].set_ylabel('Coverage Count')
    axes[0].set_title('Coverage Growth Over Time')
    axes[0].grid(True, alpha=0.3)
    axes[0].legend()

    # 2: Transitions plot
    axes[1].plot(iterations, transitions, 'r-', linewidth=2, label='Transitions')
    axes[1].set_xlabel('Iteration')
    axes[1].set_ylabel('Transition Count')
    axes[1].set_title('Transitions Growth Over Time')
    axes[1].grid(True, alpha=0.3)
    axes[1].legend()

    # 3: Coverage vs Transitions scatter
    scatter = axes[2].scatter(coverages, transitions, alpha=0.6, c=iterations, cmap='viridis')
    axes[2].set_xlabel('Coverage Count')
    axes[2].set_ylabel('Transition Count')
    axes[2].set_title('Coverage vs Transitions (colored by iteration)')
    axes[2].grid(True, alpha=0.3)
    
    if hasattr(axes[2], 'colorbar'):
        axes[2].colorbar.remove()
    colorbar = plt.colorbar(scatter, ax=axes[2])
    colorbar.set_label('Iteration')
    axes[2].colorbar = colorbar

    # 4: Trace types bar chart
    trace_types = ['Random Traces', 'Mutated Traces']
    trace_counts = [random_traces, mutated_traces]
    bars = axes[3].bar(trace_types, trace_counts, color=['skyblue', 'orange'])
    axes[3].set_ylabel('Trace Count')
    axes[3].set_title('Trace Type Summary')
    for bar, count in zip(bars, trace_counts):
        axes[3].text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.5, 
                     str(count), ha='center', va='bottom')

    # 5: Code coverage plot
    if code_coverage:
        axes[4].plot(iterations, code_coverage, 'g-', linewidth=2, label='Code Coverage')
        axes[4].set_xlabel('Iteration')
        axes[4].set_ylabel('Code Coverage')
        axes[4].set_title('Code Coverage Over Time')
        axes[4].grid(True, alpha=0.3)
        axes[4].legend()

    plt.tight_layout()
    plt.draw() 
    
    return True

if __name__ == "__main__":
    print("Starting live graph updates")
    try:
        while True:
            if update_plots():
                plt.pause(10)
            else:
                time.sleep(10)
                
    except KeyboardInterrupt:
        print("\nStopping live updates...")
    finally:
        plt.close('all')
        plt.ioff()