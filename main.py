import matplotlib.pyplot as plt
import json
import numpy as np

# Load data from rewards.json
with open('rewards.json', 'r') as f:
    data = json.load(f)

rewards = [d['reward'] / 1e18 for d in data]  # Convert to ETH

# Identify rewards for blocks that follow an empty block
follow_empty_rewards = [data[i+1]['reward'] / 1e18 for i, d in enumerate(data[:-1]) if d['txCount'] == 0]

# Calculate average rewards
avg_all_blocks = np.mean(rewards)
avg_follow_empty = np.mean(follow_empty_rewards)

# Bar chart for average rewards
plt.figure(figsize=(10, 6))
plt.bar(['All Blocks', 'Follow Empty Blocks'], [avg_all_blocks, avg_follow_empty], color=['blue', 'red'])
plt.ylabel('Average Reward (ETH)')
plt.title('Average Block Rewards')
plt.show()

# Boxplot for reward distribution
plt.figure(figsize=(10, 6))
plt.boxplot([rewards, follow_empty_rewards], labels=['All Blocks', 'Follow Empty Blocks'])
plt.ylabel('Reward (ETH)')
plt.title('Block Reward Distribution')
plt.show()
