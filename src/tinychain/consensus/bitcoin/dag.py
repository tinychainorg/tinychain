# Database.
# Block DAG data structure.
# Includes metadata.

from sqlalchemy import Column, Integer, String, ForeignKey, Float
from sqlalchemy.ext.declarative import declarative_base

from .common import Base

class DAGBlock(Base):
    __tablename__ = 'dag_blocks'

    # === Base block details. === #
    txs = Column(String)
    timestamp = Column(Float, default=0.0)
    difficulty_target = Column(String, default="0")
    nonce = Column(Integer, default=0)

    # === DAG metadata. === #
    blockhash = Column(String, primary_key=True)
    height = Column(Integer, default=0)
    acc_work = Column(String, default="0")
    parent = relationship("DAGBlock", remote_side=[blockhash], backref='child', uselist=False)
    parent_blockhash = Column(String, ForeignKey('dag_blocks.blockhash'), nullable=True) 
    # child = inferred

    def to_block(self):
        b = Block(
            str_to_u256(self.parent_blockhash),
            []
        )
        b.timestamp = self.timestamp
        b.difficulty_target = int(self.difficulty_target)
        b.nonce = self.nonce
        return b
