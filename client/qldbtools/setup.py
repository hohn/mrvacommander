from setuptools import setup, find_packages
import glob

setup(
    name='qldbtools',
    version='0.1.0',
    description='A Python package for working with CodeQL databases',
    author='Michael Hohn',
    author_email='hohn@github.com',
    packages=['qldbtools'],
    install_requires=[],
    scripts=glob.glob("bin/mc-*"),
)
