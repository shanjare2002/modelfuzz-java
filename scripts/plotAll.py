import json
import matplotlib.pyplot as plt
import numpy as np
import os
import glob
from pathlib import Path

def load_stats_from_folder(folder_path):
    """Load stats.json from a specific folder"""
    stats_file = os.path.join(folder_path, 'stats.json')
    try:
        with open(stats_file, 'r') as f:
            data = json.load(f)
        second_to_last = os.path.basename(os.path.dirname(folder_path))
        return data, second_to_last
    except Exception as e:
        print(f"Error loading {stats_file}: {e}")
        return None, None

def find_all_results_folders(base_path):
    """Find all folders containing stats.json files"""
    results = []
    
    # Look for stats.json files in subdirectories
    for stats_file in glob.glob(os.path.join(base_path, "**/stats.json"), recursive=True):
        folder_path = os.path.dirname(stats_file)
        results.append(folder_path)
    
    return sorted(results)

def find_project_root():
    """Find the project root by looking for finalOutputs directory"""
    current_dir = Path.cwd()
    
    # Check current directory and parent directories
    for path in [current_dir] + list(current_dir.parents):
        finalOutputs_path = path / "finalOutputs"
        if finalOutputs_path.exists() and finalOutputs_path.is_dir():
            return path
    
    # If not found, assume current directory is the project root
    return current_dir

def plot_all_results():
    """Plot and save each result graph separately in finalOutputs/graphs."""
    
    # Find project root and set paths
    project_root = find_project_root()
    search_path = project_root / "finalOutputs"
    graphs_dir = project_root / "finalOutputs" / "graphs"
    
    print(f"Project root: {project_root}")
    print(f"Searching for results in: {search_path}")
    print(f"Graphs will be saved to: {graphs_dir}")
    
    if not search_path.exists():
        print(f"Error: finalOutputs directory not found at {search_path}")
        return
    
    result_folders = find_all_results_folders(str(search_path))
    if not result_folders:
        print(f"No stats.json files found in {search_path}")
        return

    all_data, all_labels = [], []
    for folder in result_folders:
        data, label = load_stats_from_folder(folder)
        if data is not None:
            all_data.append(data)
            all_labels.append(label)

    if not all_data:
        print("No valid data found!")
        return

    has_code_coverage = any('CodeCoverage' in data and data['CodeCoverage'] for data in all_data)
    colors = plt.cm.tab10(np.linspace(0, 1, len(all_data)))

    # Create graphs directory
    graphs_dir.mkdir(parents=True, exist_ok=True)

    def save_plot(fig, name):
        fig.tight_layout()
        fig.savefig(graphs_dir / f"{name}.pdf")
        plt.close(fig)

    # --- Plot 1: Coverage Growth Over Time ---
    fig, ax = plt.subplots(figsize=(8, 6))
    for i, (data, label) in enumerate(zip(all_data, all_labels)):
        ax.plot(data['Coverages'], label=label, color=colors[i], linewidth=2, alpha=0.8)
    ax.set_title("Coverage Growth Over Time")
    ax.set_xlabel("Iteration")
    ax.set_ylabel("Coverage Count")
    ax.legend()
    ax.grid(True, alpha=0.3)
    save_plot(fig, "coverage_growth")

    # --- Plot 2: Transitions Growth Over Time ---
    fig, ax = plt.subplots(figsize=(8, 6))
    for i, (data, label) in enumerate(zip(all_data, all_labels)):
        ax.plot(data['Transitions'], label=label, color=colors[i], linewidth=2, alpha=0.8)
    ax.set_title("Transitions Growth Over Time")
    ax.set_xlabel("Iteration")
    ax.set_ylabel("Transition Count")
    ax.legend()
    ax.grid(True, alpha=0.3)
    save_plot(fig, "transitions_growth")

    # --- Plot 3: Final Coverage vs Transitions ---
    x = np.arange(len(all_labels))
    bar_width = 0.4
    fig, ax = plt.subplots(figsize=(8, 6))
    final_coverages = [data['Coverages'][-1] if data['Coverages'] else 0 for data in all_data]
    final_transitions = [data['Transitions'][-1] if data['Transitions'] else 0 for data in all_data]
    ax.bar(x - bar_width/2, final_coverages, bar_width, label='Coverage', color='skyblue')
    ax.bar(x + bar_width/2, final_transitions, bar_width, label='Transitions', color='orange')
    ax.set_xticks(x)
    ax.set_xticklabels(all_labels, rotation=45, ha='right')
    ax.set_title("Final Coverage and Transitions")
    ax.set_ylabel("Count")
    ax.legend()
    for i in range(len(x)):
        ax.text(x[i] - bar_width/2, final_coverages[i] + 0.5, str(final_coverages[i]), ha='center')
        ax.text(x[i] + bar_width/2, final_transitions[i] + 0.5, str(final_transitions[i]), ha='center')
    save_plot(fig, "final_coverage_vs_transitions")

    # --- Plot 4: Trace Types Summary ---
    fig, ax = plt.subplots(figsize=(8, 6))
    random_totals = [data.get('RandomTraces', 0) for data in all_data]
    mutated_totals = [data.get('MutatedTraces', 0) for data in all_data]
    ax.bar(x - bar_width/2, random_totals, bar_width, label='Random Traces', color='skyblue')
    ax.bar(x + bar_width/2, mutated_totals, bar_width, label='Mutated Traces', color='orange')
    ax.set_xticks(x)
    ax.set_xticklabels(all_labels, rotation=45, ha='right')
    ax.set_title("Trace Types Summary")
    ax.set_ylabel("Trace Count")
    ax.legend()
    for i in range(len(x)):
        ax.text(x[i] - bar_width/2, random_totals[i] + 0.5, str(random_totals[i]), ha='center')
        ax.text(x[i] + bar_width/2, mutated_totals[i] + 0.5, str(mutated_totals[i]), ha='center')
    save_plot(fig, "trace_types_summary")

    # --- Plot 5: Code Coverage (if available) ---
    if has_code_coverage:
        fig, ax = plt.subplots(figsize=(8, 6))
        for i, (data, label) in enumerate(zip(all_data, all_labels)):
            code_coverage = data.get('CodeCoverage', [])
            if code_coverage:
                ax.plot(code_coverage, label=label, color=colors[i], linewidth=2, alpha=0.8)
        ax.set_title("Code Coverage Over Time")
        ax.set_xlabel("Iteration")
        ax.set_ylabel("Code Coverage")
        ax.legend()
        ax.grid(True, alpha=0.3)
        save_plot(fig, "code_coverage")

    # --- Summary Statistics ---
    print(f"\n{'='*60}")
    print("SUMMARY STATISTICS")
    print(f"{'='*60}")
    print(f"{'Experiment':<20} {'Iterations':<12} {'Final Cov':<12} {'Final Trans':<12} {'Random':<10} {'Mutated':<10}")
    print("-" * 76)
    for data, label in zip(all_data, all_labels):
        iterations = len(data['Coverages'])
        final_cov = data['Coverages'][-1] if data['Coverages'] else 0
        final_trans = data['Transitions'][-1] if data['Transitions'] else 0
        random_traces = data.get('RandomTraces', 0)
        mutated_traces = data.get('MutatedTraces', 0)
        print(f"{label:<20} {iterations:<12} {final_cov:<12} {final_trans:<12} {random_traces:<10} {mutated_traces:<10}")
    
    print(f"\nGraphs saved to: {graphs_dir}")

if __name__ == "__main__":
    # Main comparison plots
    print("Generating main comparison plots...")
    plot_all_results()