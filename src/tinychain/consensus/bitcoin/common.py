
import sqlite3
from sqlalchemy import Column, Integer, String, ForeignKey, Float
from sqlalchemy.orm import relationship
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

Base = declarative_base()

def get_database(name, memory=False):
    DATABASE_URI = f'sqlite:///bitcoin_{name}.db'
    if memory:
        DATABASE_URI = 'sqlite:///:memory:'
    engine = create_engine(DATABASE_URI)
    Base.metadata.create_all(engine)
    Session = sessionmaker(bind=engine)
    return Session