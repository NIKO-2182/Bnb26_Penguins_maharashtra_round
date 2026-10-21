# FILE: /home/niko/BnB/Bnb26_Penguins_maharashtra_round/docs/architecture_diagram.py
# PURPOSE: Generate architecture diagram PNG for the speech summarization report
# INPUTS / OUTPUTS: PNG at /home/niko/BnB/Bnb26_Penguins_maharashtra_round/docs/architecture.png
# DEPENDS ON: matplotlib
# USED BY: Nihar Kore
# RULES: N/A
# DO NOT: N/A

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import numpy as np

fig, ax = plt.subplots(1, 1, figsize=(12, 8))
ax.set_xlim(0, 10)
ax.set_ylim(0, 8)
ax.axis('off')

# Title
ax.text(5, 7.5, "Speech Summarization using Multimodal LLMs\nProposed Architecture",
        ha='center', va='top', fontsize=16, fontweight='bold',
        fontfamily='serif')

# Define blocks and their positions
blocks = [
    {"name": "Audio Input", "desc": "Raw waveform, segmented utterances, context metadata",
     "pos": (5, 6.5), "color": "#E8F4F8"},
    {"name": "Multimodal Backbone", "desc": "Audio encoder + LLM joint processing",
     "pos": (5, 5.2), "color": "#D4EDDA"},
    {"name": "PEFT / LoRA Adaptation", "desc": "Low-rank adapters (ΔW = B × A)",
     "pos": (5, 3.9), "color": "#FFF3CD"},
    {"name": "SFT → DPO", "desc": "Supervised fine-tuning → Preference optimization",
     "pos": (5, 2.6), "color": "#D1ECF1"},
    {"name": "Evaluation", "desc": "GPT-4.1 preference + 1-7 scoring",
     "pos": (5, 1.3), "color": "#F8D7DA"},
]

# Draw blocks
for block in blocks:
    x, y = block["pos"]
    
    # Draw rounded rectangle
    box = mpatches.FancyBboxPatch(
        (x - 3, y - 0.35), 6, 0.7,
        boxstyle="round,pad=0.1",
        linewidth=2, edgecolor='#333333', facecolor=block["color"]
    )
    ax.add_patch(box)
    
    # Block name
    ax.text(x, y, block["name"], ha='center', va='center',
            fontsize=13, fontweight='bold', fontfamily='serif')
    
    # Description
    ax.text(x, y - 0.5, block["desc"], ha='center', va='top',
            fontsize=8.5, fontfamily='serif', color='#555555')
    
    # Arrow to next block
    if block["name"] != "Evaluation":
        ax.annotate('', xy=(5, y - 0.85), xytext=(5, y - 0.55),
                    arrowprops=dict(arrowstyle='->', color='#333333', lw=1.5))

# Model configurations box (side panel)
ax.text(0.5, 6.5, "Four Model Configurations",
        ha='center', va='top', fontsize=11, fontweight='bold',
        fontfamily='serif', color='#2c3e50')

configs = [
    ("Phi-4 (5.6B)\nStandard LoRA", "#e8daef"),
    ("MOSS-Audio (4.6B)\nPiSSA", "#d1ecf1"),
    ("Gemma-3n (4B)\nrsLoRA", "#fff3cd"),
    ("Qwen3 (4.2B)\nAdaLoRA", "#d4edda"),
]

for i, (cfg, color) in enumerate(configs):
    box = mpatches.FancyBboxPatch(
        (0.1, 5.8 - i*0.7), 0.8, 0.6,
        boxstyle="round,pad=0.05",
        linewidth=1.5, edgecolor='#333333', facecolor=color
    )
    ax.add_patch(box)
    ax.text(0.5, 5.8 - i*0.7 + 0.3, cfg, ha='center', va='center',
            fontsize=8, fontfamily='serif')

# Dataset info
ax.text(8.5, 6.5, "Evaluation Datasets",
        ha='center', va='top', fontsize=11, fontweight='bold',
        fontfamily='serif', color='#2c3e50')

datasets = [
    ("AMI", "Meeting/Conversational"),
    ("FLORAS", "Multilingual"),
    ("Podcast", "Long-form Narrative"),
]

for i, (ds, desc) in enumerate(datasets):
    box = mpatches.FancyBboxPatch(
        (7.7, 5.8 - i*0.6), 1.6, 0.5,
        boxstyle="round,pad=0.05",
        linewidth=1.5, edgecolor='#333333', facecolor='#e2f0cb'
    )
    ax.add_patch(box)
    ax.text(8.5, 5.8 - i*0.6 + 0.25, f"{ds}\n{desc}", ha='center', va='center',
            fontsize=8, fontfamily='serif')

# Scoring dimensions
ax.text(5, 0.5, "Scoring: Detail Accuracy | Coherence | Writing Style | Hallucination | Instruction Adherence (1-7)",
        ha='center', va='center', fontsize=8.5, fontfamily='serif',
        style='italic', color='#666666')

plt.tight_layout()
plt.savefig('/home/niko/BnB/Bnb26_Penguins_maharashtra_round/docs/architecture.png',
            dpi=150, bbox_inches='tight', facecolor='white')
print("Architecture diagram saved to: /home/niko/BnB/Bnb26_Penguins_maharashtra_round/docs/architecture.png")
