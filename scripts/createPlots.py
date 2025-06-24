import json
import matplotlib.pyplot as plt
import numpy as np
import os
import glob
from pathlib import Path

plt.rcParams.update({
    'font.size': 20,
    'axes.titlesize': 20,
    'axes.labelsize': 20,
    'xtick.labelsize': 20,
    'ytick.labelsize': 20,
    'legend.fontsize': 20,
    'figure.titlesize': 20
})

def load_stats_from_folder(folder_path):
    stats_file = os.path.join(folder_path, 'stats.json')
    try:
        with open(stats_file, 'r') as f:
            data = json.load(f)
        folder_name = os.path.basename(os.path.dirname(folder_path))
        return data, folder_name
    except Exception as e:
        print(f"Error loading {stats_file}: {e}")
        return None, None

def find_all_results_folders(base_path):
    results = []
    
    for stats_file in glob.glob(os.path.join(base_path, "**/stats.json"), recursive=True):
        folder_path = os.path.dirname(stats_file)
        results.append(folder_path)
    
    return sorted(results)

def find_project_root():
    current_dir = Path.cwd()
    
    for path in [current_dir] + list(current_dir.parents):
        finalOutputs_path = path / "finalOutputs"
        if finalOutputs_path.exists() and finalOutputs_path.is_dir():
            return path
    
    return current_dir

def plot_results_for_subdirectory(subdirectory_path):
    subdirectory_name = subdirectory_path.name
    print(f"\nProcessing subdirectory: {subdirectory_name}")
    
    result_folders = find_all_results_folders(str(subdirectory_path))
    if not result_folders:
        print(f"  No stats.json files found in {subdirectory_path}")
        return False
    
    print(f"  Found {len(result_folders)} result folders")
    
    all_data, all_labels = [], []
    for folder in result_folders:
        data, label = load_stats_from_folder(folder)
        if data is not None:
            all_data.append(data)
            all_labels.append(label)
    
    if not all_data:
        print(f"  No valid data found in {subdirectory_name}")
        return False
    
    graphs_dir = subdirectory_path / "graphs"
    graphs_dir.mkdir(parents=True, exist_ok=True)
    has_code_coverage = any('CodeCoverage' in data and data['CodeCoverage'] for data in all_data)
    colors = plt.cm.tab10(np.linspace(0, 1, len(all_data)))
    
    def save_plot(fig, name):
        fig.tight_layout()
        fig.savefig(graphs_dir / f"{name}.pdf", dpi=300, bbox_inches='tight')
        plt.close(fig)
    
    # 1: Coverage Growth Over Time
    if any(data.get('Coverages') for data in all_data):
        fig, ax = plt.subplots(figsize=(12, 8))
        for i, (data, label) in enumerate(zip(all_data, all_labels)):
            if data.get('Coverages'):
                ax.plot(data['Coverages'], label=label, color=colors[i], linewidth=3, alpha=0.8)
        ax.set_title(f"Coverage Growth Over Time - {subdirectory_name}")
        ax.set_xlabel("Iteration")
        ax.set_ylabel("Coverage Count")
        ax.legend(frameon=True, fancybox=True, shadow=True)
        ax.grid(True, alpha=0.3)
        save_plot(fig, "coverage_growth")
    
    # 2: Transitions Growth Over Time
    if any(data.get('Transitions') for data in all_data):
        fig, ax = plt.subplots(figsize=(12, 8))
        for i, (data, label) in enumerate(zip(all_data, all_labels)):
            if data.get('Transitions'):
                ax.plot(data['Transitions'], label=label, color=colors[i], linewidth=3, alpha=0.8)
        ax.set_title(f"Transitions Growth Over Time - {subdirectory_name}")
        ax.set_xlabel("Iteration")
        ax.set_ylabel("Transition Count")
        ax.legend(frameon=True, fancybox=True, shadow=True)
        ax.grid(True, alpha=0.3)
        save_plot(fig, "transitions_growth")
    
    # 3: Final Coverage vs Transitions
    x = np.arange(len(all_labels))
    bar_width = 0.4
    fig, ax = plt.subplots(figsize=(14, 10))
    final_coverages = [data['Coverages'][-1] if data.get('Coverages') else 0 for data in all_data]
    final_transitions = [data['Transitions'][-1] if data.get('Transitions') else 0 for data in all_data]

    ax.bar(x - bar_width/2, final_coverages, bar_width, label='Coverage', color='skyblue', alpha=0.8)
    ax.bar(x + bar_width/2, final_transitions, bar_width, label='Transitions', color='orange', alpha=0.8)
    ax.set_xticks(x)
    ax.set_xticklabels(all_labels, rotation=45, ha='right')
    ax.set_title(f"Final Coverage and Transitions - {subdirectory_name}")
    ax.set_ylabel("Count")
    ax.legend(frameon=True, fancybox=True, shadow=True, loc='lower right')

    y_max = max(final_coverages + final_transitions)
    ax.set_ylim(0, y_max * 1.15)
    for i in range(len(x)):
        if final_coverages[i] > 0:
            ax.text(x[i] - bar_width/2, final_coverages[i] + y_max * 0.01, 
                    str(final_coverages[i]), ha='center', va='bottom', fontsize=20)
        if final_transitions[i] > 0:
            ax.text(x[i] + bar_width/2, final_transitions[i] + y_max * 0.01, 
                    str(final_transitions[i]), ha='center', va='bottom', fontsize=20)

    save_plot(fig, "final_coverage_vs_transitions")
    
    # 4: Trace Types
    if any(data.get('RandomTraces', 0) + data.get('MutatedTraces', 0) > 0 for data in all_data):
        fig, ax = plt.subplots(figsize=(14, 10))
        random_totals = [data.get('RandomTraces', 0) for data in all_data]
        mutated_totals = [data.get('MutatedTraces', 0) for data in all_data]
        
        ax.bar(x - bar_width/2, random_totals, bar_width, label='Random Traces', color='skyblue', alpha=0.8)
        ax.bar(x + bar_width/2, mutated_totals, bar_width, label='Mutated Traces', color='orange', alpha=0.8)
        ax.set_xticks(x)
        ax.set_xticklabels(all_labels, rotation=45, ha='right')
        ax.set_title(f"Trace Types Summary - {subdirectory_name}")
        ax.set_ylabel("Trace Count")
        ax.legend(frameon=True, fancybox=True, shadow=True)
        
        for i in range(len(x)):
            if random_totals[i] > 0:
                ax.text(x[i] - bar_width/2, random_totals[i] + max(random_totals + mutated_totals) * 0.01, 
                       str(random_totals[i]), ha='center', va='bottom', fontsize=12)
            if mutated_totals[i] > 0:
                ax.text(x[i] + bar_width/2, mutated_totals[i] + max(random_totals + mutated_totals) * 0.01, 
                       str(mutated_totals[i]), ha='center', va='bottom', fontsize=12)
        
        save_plot(fig, "trace_types_summary")
    
    # 5: Code Coverage
    if has_code_coverage:
        fig, ax = plt.subplots(figsize=(12, 8))
        for i, (data, label) in enumerate(zip(all_data, all_labels)):
            code_coverage = data.get('CodeCoverage', [])
            if code_coverage:
                ax.plot(code_coverage, label=label, color=colors[i], linewidth=3, alpha=0.8)
        ax.set_title(f"Code Coverage Over Time - {subdirectory_name}")
        ax.set_xlabel("Iteration")
        ax.set_ylabel("Code Coverage")
        ax.legend(frameon=True, fancybox=True, shadow=True)
        ax.grid(True, alpha=0.3)
        save_plot(fig, "code_coverage")
    
    # Print summary
    print(f"\n  SUMMARY STATISTICS for {subdirectory_name}")
    print(f"  {'='*60}")
    print(f"  {'Experiment':<20} {'Iterations':<12} {'Final Cov':<12} {'Final Trans':<12} {'Random':<10} {'Mutated':<10}")
    print(f"  {'-' * 76}")
    for data, label in zip(all_data, all_labels):
        iterations = len(data.get('Coverages', []))
        final_cov = data['Coverages'][-1] if data.get('Coverages') else 0
        final_trans = data['Transitions'][-1] if data.get('Transitions') else 0
        random_traces = data.get('RandomTraces', 0)
        mutated_traces = data.get('MutatedTraces', 0)
        print(f"  {label:<20} {iterations:<12} {final_cov:<12} {final_trans:<12} {random_traces:<10} {mutated_traces:<10}")
    
    print(f"  Graphs saved to: {graphs_dir}")
    return True

def process_all_subdirectories():
    """Process all subdirectories in finalOutputs"""
    
    project_root = find_project_root()
    finalOutputs_path = project_root / "finalOutputs"
    
    if not finalOutputs_path.exists():
        print(f"Error: finalOutputs directory not found at {finalOutputs_path}")
        return
    
    subdirectories = [d for d in finalOutputs_path.iterdir() if d.is_dir()]
    
    if not subdirectories:
        print(f"No subdirectories found in {finalOutputs_path}")
        return
    
    print(f"Found {len(subdirectories)} subdirectories to process")
    
    processed_count = 0
    for subdirectory in sorted(subdirectories):
        if subdirectory.name.startswith('.'):
            continue
            
        success = plot_results_for_subdirectory(subdirectory)
        if success:
            processed_count += 1
    
    print(f"\n{'='*60}")
    print(f"PROCESSING COMPLETE")
    print(f"{'='*60}")
    print(f"Successfully processed {processed_count} out of {len(subdirectories)} subdirectories")

if __name__ == "__main__":
    print("Generating graphs for all subdirectories in finalOutputs...")
    process_all_subdirectories()