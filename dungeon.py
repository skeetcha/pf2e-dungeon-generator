import requests
from PIL import Image
import argparse
import time
from bs4 import BeautifulSoup
from io import BytesIO
import dukpy
import json
import base64
import random

IMAGEURL = 'https://donjon.bin.sh/fantasy/dungeon/construct.cgi?name={name}&level={level}&infest=&n_pc={nPC}&motif=&seed={seed}&dungeon_size={dungeonSize}&map_cols={mapCols}&map_rows={mapRows}&dungeon_layout={dungeonLayout}&peripheral_egress={egress}&room_layout={roomLayout}&room_size={roomSize}&room_polymorph={polymorph}&door_set={doors}&corridor_layout={corridors}&remove_deadends={deadends}&add_stairs={stairs}&image_size=&map_style={style}&grid={grid}'
STATUSURL = 'https://donjon.bin.sh/fantasy/dungeon/status.fcgi?auth={auth}&id={id}'
JSONURL = 'https://donjon.bin.sh/fantasy/dungeon/download/json.cgi?auth={auth}&id={id}'

def checkRes(res):
    if res.status_code != 200:
        raise ValueError(f'Error {res.status_code}: {res.reason}')

def infest(level, motif, playerNum, rooms):
    # Get Monsters
    res = requests.get('https://mimic-fight-club.github.io/monsterTable-2023-07-22.js')
    checkRes(res)
    interpreter = dukpy.JSInterpreter()
    interpreter.evaljs(res.text)
    monsters = json.loads(interpreter.evaljs('JSON.stringify(creatureList)'))

    for i in range(len(rooms)):
        difficulty = random.randint(0, 4)
        xpBudget = (60 + ((playerNum - 4) * 20)) if difficulty == 0 else (80 + ((playerNum - 4) * 20)) if difficulty == 1 else (120 + ((playerNum - 4) * 30)) if difficulty == 2 else (160 + ((playerNum - 4) * 40))
    
    return rooms

def main(args):
    res = requests.get(IMAGEURL.format(name=args.name, level=args.level, seed=args.seed, dungeonSize=args.dungeonSize, mapCols=args.dungeonCols if args.dungeonCols else '', mapRows=args.dungeonRows if args.dungeonRows else '', dungeonLayout=args.dungeonLayout, egress=args.egress, roomLayout=args.roomLayout, roomSize=args.roomSize, polymorph=args.polymorph, doors=args.doors, corridors=args.corridors, deadends=args.deadends, stairs=args.stairs, style=args.style, grid=args.grid, nPC=args.playerNum))
    dungeonData = res.json()
    res = requests.get(STATUSURL.format(auth=dungeonData['auth'], id=dungeonData['id']))
    checkRes(res)
    statusData = res.json()

    while 'note' in statusData.keys():
        time.sleep(1)
        res = requests.get(STATUSURL.format(auth=dungeonData['auth'], id=dungeonData['id']))
        checkRes(res)
        statusData = res.json()
    
    assert 'done' in statusData.keys() and statusData['done'] == 1
    soup = BeautifulSoup(statusData['html'], 'html.parser')
    images = list(map(lambda x: x['src'], soup.find_all('img')))
    res = requests.get('https://donjon.bin.sh' + images[0])
    checkRes(res)
    output = BytesIO()
    output.write(res.content)
    mapImage = Image.open(output)
    output = BytesIO()
    mapImage.save(output, format='PNG')
    output.seek(0)
    images[0] = base64.b64encode(output.read()).decode('utf-8')
    mapImage.close()
    output.close()
    res = requests.get('https://donjon.bin.sh' + images[1])
    checkRes(res)
    output = BytesIO()
    output.write(res.content)
    keyImage = Image.open(output)
    output = BytesIO()
    keyImage.save(output, format='PNG')
    output.seek(0)
    images[1] = base64.b64encode(output.read()).decode('utf-8')
    keyImage.close()
    output.close()
    res = requests.get(JSONURL.format(auth=dungeonData['auth'], id=dungeonData['id']))
    checkRes(res)
    res = requests.get('https://donjon.bin.sh' + res.json()['href'])
    checkRes(res)
    mapData = res.json()
    random.seed(args.seed)
    mapData['rooms'] = infest(args.level, args.motif, args.playerNum, mapData['rooms'][1:])

def parseArgs():
    parser = argparse.ArgumentParser(description='A random dungeon generator for Pathfinder 2nd edition')
    infoGroup = parser.add_argument_group('Dungeon Info')
    infoGroup.add_argument('--name', type=str, help='The name of the dungeon')
    infoGroup.add_argument('--level', type=int, default=1, help='The level of the dungeon')
    infoGroup.add_argument('--npc', type=int, dest='playerNum', help='The number of players.', default=4)
    infoGroup.add_argument('--details', type=str, choices=['None', 'Basic'], help='Details to add to the dungeon', default='None')
    infoGroup.add_argument('--motif', type=str, choices=['None', 'Abandoned', 'Aberrant', 'Giant', 'Undead', 'Vermin', 'Aquatic', 'Desert', 'Underdark', 'Arcane', 'Fire', 'Cold', 'Abyssal', 'Infernal'], help='The motif of the dungeon', default='None')
    dungeonSettings = parser.add_argument_group('Dungeon Settings')
    dungeonSettings.add_argument('--seed', type=str, help='The seed for the dungeon')
    dungeonSettings.add_argument('--dsize', type=str, dest='dungeonSize', help='The size of the dungeon', choices=['Fine', 'Diminutive', 'Tiny', 'Small', 'Medium', 'Large', 'Huge', 'Gargantuan', 'Colossal', 'Custom'], default='Medium')
    dungeonSettings.add_argument('--dcols', type=int, dest='dungeonCols', help='The amount of columns in the dungeon. Used for Custom dungeon size.')
    dungeonSettings.add_argument('--drows', type=int, dest='dungeonRows', help='The amount of rows in the dungeon. Used for Custom dungeon size.')
    dungeonSettings.add_argument('--dlayout', type=str, dest='dungeonLayout', help='The layout of the dungeon', choices=['Square', 'Rectangle', 'Box', 'Cross', 'Dagger', 'Saltire', 'Keep', 'Hexagon', 'Round', 'Cavernous'], default='Rectangle')
    dungeonSettings.add_argument('--egress', type=str, help='Add egress(es) to the dungeon?', choices=['No', 'Yes', 'Many', 'Tiling'], default='No')
    roomSettings = parser.add_argument_group('Room Settings')
    roomSettings.add_argument('--rlayout', type=str, dest='roomLayout', help='The layout of the rooms', choices=['Sparse', 'Scattered', 'Dense', 'Symmetric'], default='Scattered')
    roomSettings.add_argument('--rsize', type=str, dest='roomSize', help='The size of the rooms', choices=['Small', 'Medium', 'Large', 'Huge', 'Gargantuan', 'Colossal'], default='Medium')
    roomSettings.add_argument('--polymorph', type=str, help='Change the shapes of the rooms?', choices=['No', 'Yes', 'Many'], default='Yes')
    roomSettings.add_argument('--doors', type=str, help='Which set of doors to use?', choices=['None', 'Basic', 'Secure', 'Standard', 'Deathtrap'], default='Standard')
    roomSettings.add_argument('--corridors', type=str, help='The types of corridors to use', choices=['Labyrinth', 'Errant', 'Straight'], default='Errant')
    roomSettings.add_argument('--deadends', type=str, help='Should dead-ends be removed?', choices=['None', 'Some', 'All'], default='Some')
    roomSettings.add_argument('--stairs', type=str, help='Should stairs be added?', choices=['No', 'Yes', 'Many'], default='Yes')
    mapSettings = parser.add_argument_group('Map Settings')
    mapSettings.add_argument('--style', type=str, help='The style of the map', choices=['Standard', 'Classic', 'Crosshatch', 'GraphPaper', 'Parchment', 'Marble', 'Sandstone', 'Slate', 'Aquatic', 'Infernal', 'Glacial', 'Wooden', 'Asylum', 'Steampunk', 'Gamma'], default='Standard')
    mapSettings.add_argument('--grid', type=str, help='What kind of grid to use', choices=['None', 'Square', 'Hex', 'VertHex'], default='Square')
    args = parser.parse_args()
    validateArgs(args)
    return args

def validateArgs(args):
    if args.name == None:
        res = requests.get('https://donjon.bin.sh/fantasy/random/rpc-fantasy.fcgi?type=Dungeon%20Name&n=1')
        checkRes(res)
        args.name = res.json()[0]
    
    if args.seed == None:
        args.seed = int(time.time())
    
    if args.level < 1 or args.level > 20:
        raise ValueError('Level must be between 1 and 20')

if __name__ == '__main__':
    main(parseArgs())