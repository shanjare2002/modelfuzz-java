import json
import matplotlib.pyplot as plt
import numpy as np

# Load the JSON data
with open('output/modelfuzz/stats.json', 'r') as f:
    data = json.load(f)

# Extract data
coverages = data['Coverages']
transitions = data['Transitions']
random_traces = data['RandomTraces']
mutated_traces = data['MutatedTraces']
code_coverage = data.get('CodeCoverage', None)  # Optional: Only plot if available

# Create iteration indices
iterations = range(len(coverages))

# Adjust number of subplots based on presence of CodeCoverage
plot_rows = 3 if code_coverage else 2
plot_cols = 2
fig, axes = plt.subplots(plot_rows, plot_cols, figsize=(15, 12))

# Flatten axes for easier indexing
axes = axes.flatten()

# Plot 1: Coverage over iterations
axes[0].plot(iterations, coverages, 'b-', linewidth=2, label='Coverage')
axes[0].set_xlabel('Iteration')
axes[0].set_ylabel('Coverage Count')
axes[0].set_title('Coverage Growth Over Time')
axes[0].grid(True, alpha=0.3)
axes[0].legend()

# Plot 2: Transitions over iterations
axes[1].plot(iterations, transitions, 'r-', linewidth=2, label='Transitions')
axes[1].set_xlabel('Iteration')
axes[1].set_ylabel('Transition Count')
axes[1].set_title('Transitions Growth Over Time')
axes[1].grid(True, alpha=0.3)
axes[1].legend()

# Plot 3: Coverage vs Transitions
scatter = axes[2].scatter(coverages, transitions, alpha=0.6, c=iterations, cmap='viridis')
axes[2].set_xlabel('Coverage Count')
axes[2].set_ylabel('Transition Count')
axes[2].set_title('Coverage vs Transitions (colored by iteration)')
axes[2].grid(True, alpha=0.3)
colorbar = plt.colorbar(scatter, ax=axes[2])
colorbar.set_label('Iteration')

# Plot 4: Trace type summary (bar chart)
trace_types = ['Random Traces', 'Mutated Traces']
trace_counts = [random_traces, mutated_traces]
bars = axes[3].bar(trace_types, trace_counts, color=['skyblue', 'orange'])
axes[3].set_ylabel('Trace Count')
axes[3].set_title('Trace Type Summary')
for bar, count in zip(bars, trace_counts):
    axes[3].text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.5, 
                 str(count), ha='center', va='bottom')

# Plot 5: Code Coverage over iterations (if available)
if code_coverage:
    axes[4].plot(iterations, code_coverage, 'g-', linewidth=2, label='Code Coverage')
    axes[4].set_xlabel('Iteration')
    axes[4].set_ylabel('Code Coverage')
    axes[4].set_title('Code Coverage Over Time')
    axes[4].grid(True, alpha=0.3)
    axes[4].legend()

# Hide any unused axes
for i in range(5, len(axes)):
    fig.delaxes(axes[i])

plt.tight_layout()
plt.show()

# Print summary statistics
print("=== ModelFuzz Statistics Summary ===")
print(f"Total iterations: {len(coverages)}")
print(f"Final coverage: {coverages[-1]}")
print(f"Final transitions: {transitions[-1]}")
print(f"Coverage growth: {coverages[-1] - coverages[0]} ({((coverages[-1] - coverages[0]) / coverages[0] * 100):.1f}%)")
print(f"Transition growth: {transitions[-1] - transitions[0]} ({((transitions[-1] - transitions[0]) / transitions[0] * 100):.1f}%)")
print(f"Random traces: {random_traces}")
print(f"Mutated traces: {mutated_traces}")
print(f"Total traces: {random_traces + mutated_traces}")
if code_coverage:
    print(f"Final code coverage: {code_coverage[-1]}")
    print(f"Code coverage growth: {code_coverage[-1] - code_coverage[0]} ({((code_coverage[-1] - code_coverage[0]) / code_coverage[0] * 100):.1f}%)")
