from cognize.client import AgentClient
from cognize.ai_challenge import AIChallengeClient, get_reputation, get_agent_info
from cognize.precompiles import (
    REGISTRY_ADDRESS, REPUTATION_ADDRESS, WALLET_ADDRESS,
    TRUST_BLOCKED, TRUST_UNKNOWN, TRUST_LIMITED, TRUST_FULL,
)

__version__ = "0.4.0"
__all__ = [
    "AgentClient",
    "AIChallengeClient",
    "get_reputation",
    "get_agent_info",
    "REGISTRY_ADDRESS", "REPUTATION_ADDRESS", "WALLET_ADDRESS",
    "TRUST_BLOCKED", "TRUST_UNKNOWN", "TRUST_LIMITED", "TRUST_FULL",
]
